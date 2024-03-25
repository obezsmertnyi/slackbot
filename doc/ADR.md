# Обгрунтування використаних технологій (ADR)

## Постановка завдання

- Мета: Метою цього завдання є розробка системи, яка дозволить розробникам легко і швидко управляти версіями своїх додатків на Kubernetes. Система повинна бути інтегрована з месенджером, щоб розробники могли отримувати сповіщення про розгортання нової версії свого додатку.

- Завдання: Як розробник я хочу мати можливість підключити Slack-бота до каналу команди для спрощення управління версіями додатків. Бот повинен надавати функції list, diff, promote та rollback для отримання актуального статусу версій аплікації на різних середовищах (dev, qa, staging, prod) та виконання необхідних дій.

## Вибір архітектурного рішення для Slackbot

Виходячи з запропонованого стеку у завданні (App based on GO Slack API library, FluxCD for GitOps, Applications CRD, SQLite, Mock-components), було розроблено 2 окремих технічних рішення:
- slackbot on kind, розроблений на початковому етапі (реліз v1.0.0)
- slackbot on k8s, розроблений на фінальному етапі (реліз v2.0.0)

### 1. Slackbot on kind, (реліз v1.0.0) має наступні компоненти:
 - платформа розгортання - kind
 - App Slackbot on Go 1.21
 - SQLite local db
 - FluxCD
 - GitHub (Source Code Repo & Container Registry)
 - GitHub Actions for CI/CD

### 2. Slackbot on k8s, (реліз v2.0.0) має наступні компоненти:
 - платформа розгортання - k8s
 - App Slackbot on Go 1.21
 - Docker-compose for Slackbot
 - SQLite local db
 - FluxCD
 - GitHub (Source Code Repo & Container Registry)
 - GitHub Actions for CI/CD
 - Vault secret manager
 - Prometheus metrics

Детально архітектура буде описана у наступному документі HLD.md
