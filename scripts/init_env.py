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
    print(f"Wrote {OUT_FILE}")
    print('\nYou can now run:')
    print('  $env:INTERACTIVE=\"true\"; go run ./cmd')


if __name__ == '__main__':
    main()
