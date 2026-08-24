# load-tests-cli

CLI simples para executar testes de carga em serviços web.

## Requisitos

- Go 1.25 ou superior

## Uso

Execute informando a URL, o total de requisições e a concorrência:

```bash
go run . --url https://example.com --requests 100 --concurrency 10
```

Opções:

| Flag | Atalho | Padrão | Descrição |
| --- | --- | --- | --- |
| `--url` | `-u` | vazio | URL do serviço testado |
| `--requests` | `-r` | `1` | Total de requisições |
| `--concurrency` | `-c` | `1` | Chamadas simultâneas |

O relatório exibe o total de requisições, sucessos HTTP `200`, tempo de execução e outros status ou erros.

## Build

```bash
go build -o load-tests-cli .
./load-tests-cli -u https://example.com -r 100 -c 10
```

Para executar os testes:

```bash
go test ./...
```

## Docker

```bash
docker build -t load-tests-cli .
docker run --rm load-tests-cli -u https://example.com -r 100 -c 10
```

## Contato

Desenvolvido por Luana Andrade - luanaands@gmail.com

Aproveite! 🚀
