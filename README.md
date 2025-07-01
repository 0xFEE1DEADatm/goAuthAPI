# Реализация сервиса аутентификации на Go с использованием PostgreSQL и Docker

## Требования к токенам

### Access Token
- Формат: JWT
- Алгоритм подписи: SHA512
- Хранение: только на клиенте (в базе не хранится)

### Refresh Token
- Формат: произвольный (передается в base64)
- Хранение: только bcrypt-хеш в базе
- Защита: 
  - от повторного использования
  - от клиентских модификаций

## Правила работы refresh-операции

1. Обновление возможно только для исходной пары токенов
2. При изменении User-Agent:
   - операция блокируется
   - пользователь автоматически разлогинивается
3. При смене IP:
   - отправляется webhook-уведомление
   - операция разрешена
---

## Запуск
docker-compose -f docker-compose.yml up -d

## Swagger

После запуска документация доступна по адресу:
http://localhost:8080/swagger/index.html

## Эндпоинты

### POST /tokens
Запрос:
{
  "user_guid": "1c9c3b2e-8a7f-4f0d-9551-341234567890"
}
Ответ:
{
    "access_token": "eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2d1aWQiOiIxYzljM2IyZS04YTdmLTRmMGQtOTU1MS0zNDEyMzQ1Njc4OTAiLCJleHAiOjE3NTEzNjY0NjYsImlhdCI6MTc1MTM2NTU2Nn0.0j6oTMGXNoJWbxwxpajxVFH70gebzRRhOmJZ3oDEZoYdT3S4Lm3LMb-7nInif3kuZcMqnt26jeBAsmLYUNMuew",
    "refresh_token": "TQe99uZHZk5B0SwsQ/NX0yEAOHjiDSO1n3f5HIjzePM="
}

### POST /tokens/refresh
Запрос:
{
  "user_guid": "1c9c3b2e-8a7f-4f0d-9551-341234567890",
  "refresh_token": "TQe99uZHZk5B0SwsQ/NX0yEAOHjiDSO1n3f5HIjzePM="
}
Ответ:
{
    "access_token": "eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2d1aWQiOiIxYzljM2IyZS04YTdmLTRmMGQtOTU1MS0zNDEyMzQ1Njc4OTAiLCJleHAiOjE3NTEzNjY0OTIsImlhdCI6MTc1MTM2NTU5Mn0.TxmqMh8S5zNoX0k-SGPwzeHVu4Y_Lh6A2Psz03B83OZm5tqP_Q7K3Zn38QmPaqYKuiGu0FWaoYj0YTFkxCacCA",
    "refresh_token": "hF/pSfCwGdnWCthUT39Vr8R9W3rI2p4N5BpsuQ6Ey9I="
}

### GET /me
Ответ:
{"user_guid":"1c9c3b2e-8a7f-4f0d-9551-341234567890"}

### POST /logout
Ответ:
logged out
