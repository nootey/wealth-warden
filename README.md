# Wealth Warden 👋

An open-source finance tracker focused on simplicity and usability.

## 🚀 About Wealth Warden
Wealth Warden is a personal finance tracker designed to be simple, intuitive, and efficient. Inspired by my own Excel-based template, this project aims to provide a seamless experience for tracking income, expenses, and financial goals—without unnecessary complexity.

## 🎯 Features
- Easy-to-use interface – No clutter, just what you need.
- Income & Expense Tracking – Stay on top of your cash flow.
- Budgeting Tools – Set and manage your financial goals.
- Data Visualization – Simple charts for quick insights.
- Custom Categories – Personalize your tracking system.
- Open Source – You can confirm your data is not being manipulated. 

## 🛠️ Tech Stack
Client: Vue 3 + TypeScript
State Management: Pinia
UI Framework: Primevue 4

## 📦 Installation

### Environment variables

The client requires a .env file to be present in project root. Create it and fill it out according to this template:
```js
VITE_APP_PORT=3000
VITE_APP_PRODUCTION_MODE=<value>
VITE_API_BASE_URL="<api-url>"
```

### 1️⃣ Clone the Repository
```bash
git clone https://github.com/nootey/wealth-warden-client.git
cd wealth-warden
```
### 2️⃣ Install Dependencies
```bash
npm install
```

### 3️⃣ Run the Development Server
```bash
npm run dev
```

The client should be available on: http://localhost:3000