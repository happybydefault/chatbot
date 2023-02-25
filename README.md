# chatbot

`chatbot` is a work-in-progress WhatsApp chatbot that responds with AI (OpenAI's GPT-3.5).

## Usage

1. Create an `.env` file with the following environment variables:

   ```text
   POSTGRES_USER="user"
   POSTGRES_PASSWORD="password"
   POSTGRES_DB="db"
   
   OPENAI_API_KEY="key"
   ```

2. Start services:

   ```sh
   docker compose up --detach
   ```

3. Link device to WhatsApp using the QR code that was printed in the logs:

   ```sh
   docker compose logs --follow chatbot
   ```
