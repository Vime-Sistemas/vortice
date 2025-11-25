#!/usr/bin/env python3
r"""
Interactive environment file initializer for Vortice
Creates a `.env.local` file in the repository root with answers from the user.

Usage (PowerShell):
    cd C:\Users\Parceiros\vime\lab\libs\vortice
    python .\scripts\init_env.py

Options:
    --yes    Accept defaults for all prompts (non-interactive)

The script asks for the common variables used by the project and writes
KEY=VALUE lines into `.env.local`.
"""
import argparse
import os
import textwrap
import sys
import subprocess
import threading
import time
import shutil

ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), '..'))
OUT_FILE = os.path.join(ROOT, '.env.local')

DEFAULTS = {
    'APP_PORT': '8080',
    'BACKEND_URLS': '',
    'START_LOCAL_BACKENDS': 'true',
    'LOCAL_BACKEND_COUNT': '3',
    'LOCAL_BACKEND_START_PORT': '8081',
    'LOCAL_BACKEND_FORCE': 'false',
    'LOAD_BALANCER_ALGO': 'round_robin',
    'RATE_LIMIT_RPS': '0',
    'RATE_LIMIT_BURST': '1',
    'BACKEND_RATE_LIMITS': '',
    'IP_HASH_HEADER': '',
    'INTERACTIVE': 'true'
}

ALGOS = ['round_robin', 'least_conn', 'random', 'ip_hash']


def ask(prompt, default=None, yes=False):
    if yes:
        return default if default is not None else ''
    if default is None:
        return input(f"{prompt}: ")
    v = input(f"{prompt} [{default}]: ")
    return v.strip() or default


def confirm(prompt, default=False, yes=False):
    if yes:
        return default
    d = 'Y/n' if default else 'y/N'
    v = input(f"{prompt} ({d}): ")
    if v.strip() == '':
        return default
    return v.strip().lower() in ('y', 'yes')


def write_env_file(path, values):
    lines = []
    for k, v in values.items():
        lines.append(f"{k}={v}")
    content = "\n".join(lines) + "\n"
    with open(path, 'w', encoding='utf-8') as f:
        f.write(content)
    return content


def main():
    p = argparse.ArgumentParser(description='Init .env.local for Vortice')
    p.add_argument('--yes', '-y', action='store_true', help='Accept defaults for all prompts')
    args = p.parse_args()

    yes = args.yes
    print(textwrap.dedent(f"""
    Inicializador de ambiente para Vortice
    Será criado/reescrito: {OUT_FILE}
    Pressione Ctrl+C para abortar a qualquer momento.
    """))

    app_port = ask('Qual será a porta da aplicação', DEFAULTS['APP_PORT'], yes)
    # validate integer
    try:
        int(app_port)
    except Exception:
        print('Porta inválida, usando valor padrão')
        app_port = DEFAULTS['APP_PORT']

    start_local = confirm('Iniciar backends locais para desenvolvimento?', DEFAULTS['START_LOCAL_BACKENDS'].lower() == 'true', yes)
    backend_urls = DEFAULTS['BACKEND_URLS']
    local_count = DEFAULTS['LOCAL_BACKEND_COUNT']
    local_start = DEFAULTS['LOCAL_BACKEND_START_PORT']
    local_force = DEFAULTS['LOCAL_BACKEND_FORCE']

    if start_local:
        local_count = ask('Quantos backends locais iniciar', DEFAULTS['LOCAL_BACKEND_COUNT'], yes)
        try:
            int(local_count)
        except Exception:
            print('Valor inválido para contagem, usando padrão')
            local_count = DEFAULTS['LOCAL_BACKEND_COUNT']
        local_start = ask('Porta inicial dos backends locais', DEFAULTS['LOCAL_BACKEND_START_PORT'], yes)
        try:
            int(local_start)
        except Exception:
            print('Porta inválida, usando porta padrão')
            local_start = DEFAULTS['LOCAL_BACKEND_START_PORT']
        local_force = 'true' if confirm('Forçar apenas backends locais (ignorar BACKEND_URLS)?', DEFAULTS['LOCAL_BACKEND_FORCE'].lower() == 'true', yes) else 'false'
        # ainda permitir que o usuário adicione BACKEND_URLS extras
        backend_urls = ask('BACKEND_URLS extras (separadas por vírgula) ou vazio', DEFAULTS['BACKEND_URLS'], yes)
    else:
        backend_urls = ask('BACKEND_URLS (separadas por vírgula) ex: http://host:8081,http://host:8082', DEFAULTS['BACKEND_URLS'], yes)

    print('\nAlgoritmos disponíveis:')
    for a in ALGOS:
        print(' -', a)
    algo = ask('Escolha o algoritmo', DEFAULTS['LOAD_BALANCER_ALGO'], yes)
    if algo not in ALGOS:
        print(f"Algoritmo desconhecido '{algo}', selecionando '{DEFAULTS['LOAD_BALANCER_ALGO']}'")
        algo = DEFAULTS['LOAD_BALANCER_ALGO']

    rate_rps = ask('RATE_LIMIT_RPS global (0 = desabilitado)', DEFAULTS['RATE_LIMIT_RPS'], yes)
    rate_burst = ask('RATE_LIMIT_BURST global', DEFAULTS['RATE_LIMIT_BURST'], yes)
    backend_rates = ask('Limites por backend como CSV (rps:burst;...) na ordem de BACKEND_URLS', DEFAULTS['BACKEND_RATE_LIMITS'], yes)
    ip_hash_header = ask('Header para IP hash (deixe vazio para usar RemoteAddr)', DEFAULTS['IP_HASH_HEADER'], yes)
    interactive = 'true' if confirm('Ativar console interativo (INTERACTIVE)?', DEFAULTS['INTERACTIVE'].lower()=='true', yes) else 'false'

    values = {
        'APP_PORT': app_port,
        'BACKEND_URLS': backend_urls,
        'START_LOCAL_BACKENDS': 'true' if start_local else 'false',
        'LOCAL_BACKEND_COUNT': local_count,
        'LOCAL_BACKEND_START_PORT': local_start,
        'LOCAL_BACKEND_FORCE': local_force,
        'LOAD_BALANCER_ALGO': algo,
        'RATE_LIMIT_RPS': rate_rps,
        'RATE_LIMIT_BURST': rate_burst,
        'BACKEND_RATE_LIMITS': backend_rates,
        'IP_HASH_HEADER': ip_hash_header,
        'INTERACTIVE': interactive,
    }

    print('\nPronto para gravar o seguinte em .env.local:')
    for k, v in values.items():
        print(f"{k}={v}")

    if os.path.exists(OUT_FILE) and not yes:
        if not confirm(f"O arquivo {OUT_FILE} já existe. Sobrescrever?", False, yes):
            print('Cancelado pelo usuário.')
            sys.exit(1)
    if not confirm('Gravar arquivo agora?', True, yes):
        print('Cancelado pelo usuário.')
        sys.exit(1)

    content = write_env_file(OUT_FILE, values)
    print(f"Escrito {OUT_FILE}")
    print('\nVocê pode agora executar:')
    print('  $env:INTERACTIVE=\"true\"; go run ./cmd')

    # Pergunta para iniciar o Vortice agora
    if confirm('Deseja iniciar o Vortice agora?', True, yes):
        # helper spinner
        def spinner(msg, stop_event):
            chars = ['|', '/', '-', '\\']
            i = 0
            while not stop_event.is_set():
                print(f"\r{msg} {chars[i%len(chars)]}", end='', flush=True)
                time.sleep(0.12)
                i += 1
            print('\r' + ' '*(len(msg)+4) + '\r', end='', flush=True)

        # run a command with spinner, capture output; return (rc, out, err)
        def run_cmd(cmd, cwd=None):
            stop = threading.Event()
            t = threading.Thread(target=spinner, args=(f"Executando: {' '.join(cmd)}", stop))
            t.start()
            try:
                proc = subprocess.run(cmd, cwd=cwd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
                return (proc.returncode, proc.stdout.decode('utf-8', errors='replace'), proc.stderr.decode('utf-8', errors='replace'))
            finally:
                stop.set()
                t.join()

        # ensure `go` is available
        if not shutil.which('go'):
            print('Go não encontrado no PATH. Instale o Go para prosseguir.')
            sys.exit(1)

        # 1) Run tests
        print('Executando suíte de testes...')
        rc, out, err = run_cmd(['go', 'test', './...'])
        if rc != 0:
            print('Testes falharam. Saída:')
            print(out)
            print(err)
            sys.exit(rc)
        print('Testes passaram.')

        # 2) Build
        print('Compilando o binário...')
        bin_name = 'vortice.exe' if os.name == 'nt' else 'vortice'
        rc, out, err = run_cmd(['go', 'build', '-o', bin_name, './cmd/main.go'])
        if rc != 0:
            print('Build falhou. Saída:')
            print(out)
            print(err)
            sys.exit(rc)
        print(f'Build concluído: ./{bin_name}')

        # 3) Execute
        interactive_mode = values.get('INTERACTIVE', 'false').lower() == 'true'
        bin_path = os.path.join(ROOT, bin_name)
        print('Iniciando Vortice...')
        if interactive_mode:
            # Attach to terminal so REPL works and logs are visible
            print('Modo interativo ativado — passando controle ao processo (Ctrl+C para sair).')
            try:
                # Use subprocess.Popen to run the binary, inheriting stdin/stdout/stderr
                proc = subprocess.Popen([bin_path], stdin=sys.stdin, stdout=sys.stdout, stderr=sys.stderr)
                proc.wait()
            except Exception as e:
                print(f'Erro ao executar o binário: {e}')
                sys.exit(1)
        else:
            # Run with stdout/stderr suppressed, show spinner while running
            def run_and_wait():
                with open(os.devnull, 'wb') as devnull:
                    proc = subprocess.Popen([bin_path], stdout=devnull, stderr=devnull)
                    try:
                        while proc.poll() is None:
                            time.sleep(0.5)
                    except KeyboardInterrupt:
                        proc.terminate()
                        proc.wait()

            t_run = threading.Thread(target=run_and_wait, daemon=True)
            t_run.start()
            try:
                # show a simple waiting message
                print('Vortice iniciado em modo não interativo (logs suprimidos). Pressione Ctrl+C para parar.')
                while t_run.is_alive():
                    time.sleep(0.5)
            except KeyboardInterrupt:
                print('\nInterrompendo...')
            sys.exit(0)


if __name__ == '__main__':
    main()
