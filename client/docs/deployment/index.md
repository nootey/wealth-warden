## Deployment

### Environment variables

The client requires a .env file to be present in project root. Create it and fill it out according to this template:
```js
VITE_APP_PORT=3000
VITE_APP_PRODUCTION_MODE=<value>
VITE_API_BASE_URL="<api-url>"
```

### Run the client

#### 1️⃣ Clone the Repository
```bash
git clone https://github.com/nootey/wealth-warden-client.git
cd wealth-warden
```
#### 2️⃣ Install Dependencies
```bash
npm install
```

#### 3️⃣ Run the Development Server
```bash
npm run dev
```

The client should be available on: http://localhost:3000