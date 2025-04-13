# Backend — BlogPoint
Бэкенд веб-приложения BlogPoint, написанный на Go с использованием Fiber и GORM.
В качестве базы данных используется PostgreSQL. Приложение может быть запущено с помощью Docker.

### Развёртывание приложения (через Docker Compose)
1. Клонируйте репозиторий
```cmd
git clone https://github.com/yourname/blogpoint.git
cd blogpoint
```
2. Запустите проект
```cmd
docker-compose up --build
```
3. Приложение будет доступно по адресу:
```
http://localhost:8080
```
