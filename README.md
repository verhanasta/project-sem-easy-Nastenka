# ITMO DevOps Course - Semester 1 Project

Проект представляет собой HTTP-сервер на Go для аналитики цен супермаркета с автоматизацией деплоя через bash-скрипты.

## 🛠 Технологии
- **Go** (версии 1.21+)
- **PostgreSQL** (версии 15+)
- **Bash**
- Дополнительные зависимости:
    - `github.com/lib/pq` (драйвер PostgreSQL для Go)
    - `archive/zip`, `encoding/csv` (работа с архивами и CSV)

## 📋 Требования
1. Установленный Go (≥ 1.21)
2. Установленный PostgreSQL (≥ 15)
3. Bash (для запуска скриптов)
4. curl (для тестирования запросов)

## 🚀 Запуск проекта

### 1. Клонирование репозитория
```bash
git clone https://github.com/Vald3mare/itmo-devops-sem1-project-template.git
cd itmo-devops-sem1-project-template
сd cmd/api
go run .