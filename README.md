# TokenTransfer

A Go-based GraphQL service for managing token wallets and transactions, containerized with Docker.

---

##  Project Structure

```plaintext
db/
└── init/
    └── schema.sql          # Create & initialize wallet and transaction tables
api.go                      # GraphQL schema & core functionality
api_test.go                 # Tests (via Testcontainers)
main.go                     # Starts the HTTP server
docker-compose.yml          # Docker Compose setup
Dockerfile                  # Docker image definition
go.mod                      # Module definitions
go.sum                      # Module dependencies
```

---

##  Database Schema

`db/init/schema.sql` creates the following tables on startup:

* **Wallets**
  Tracks each wallet’s address and balance.

* **TransactionHistory**
  Records every token transfer between wallets.

---

##  Getting Started

1. **Install Docker**

   ```sh
   pip install docker
   ```

2. **Clone this repository**

   ```sh
   git clone https://github.com/piotrkotlowski/TokenTransfer.git
   cd TokenTransfer
   ```

3. **Build & run**

   ```sh
   docker compose up --build
   ```

4. **Open in browser**
   Navigate to:
   `http://localhost:8080/`

---

##  GraphQL API

### 1. Visualize All Wallets

```graphql
query {
  wallets {
    balance
    address
  }
}
```

**Sample Response:**

```json
{
  "data": {
    "wallets": [
      { "balance": 100, "address": "0xAvx" },
      { "balance": 250, "address": "0xabc" }
    ]
  }
}
```

### 2. Create a New Wallet

```graphql
mutation {
  createWallet(address: "0xAvx", balance: 100) {
    address
    balance
  }
}
```

**Sample Response:**

```json
{
  "data": {
    "createWallet": { "address": "0xAvx", "balance": 100 }
  }
}
```

### 3. Make a Transaction

```graphql
mutation {
  makeTransaction(sender: "0xAvx", receiver: "0xabc", amount: 10) {
    sender
    receiver
    amount
  }
}
```

**Sample Response:**

```json
{
  "data": {
    "makeTransaction": {
      "sender": "0xAvx",
      "receiver": "0xabc",
      "amount": 10
    }
  }
}
```

### 4. View Transaction History

```graphql
query {
  transactions {
    transaction_id
    sender
    receiver
    amount
  }
}
```

**Sample Response:**

```json
{
  "data": {
    "transactions": [
      {
        "transaction_id": 1,
        "sender": "0xAvx",
        "receiver": "0xabc",
        "amount": 10
      }
    ]
  }
}
```

---

##  Tests

```sh
go test -race -v ./...
```

**Included Test Cases:**

1. **Transfer Success**
   Normal transfers of 5 tokens.

2. **Insufficient Balance**
   Attempt to send more tokens than available.

3. **Concurrent Transfers**
   Simulate multiple simultaneous transactions.

4. **SQL Injection Attempt**
   Ensure inputs are sanitized.

5. **Missing Sender/Receiver**
   Error when sender or receiver not provided.

6. **Duplicate Wallet**
   Creating a wallet that already exists.

7. **Negative Amount**
   Reject transfers where `amount < 0`.

**Example Errors:**

* **Insufficient Funds**

  ```json
  {
    "errors": [
      { "message": "Insufficient balance" }
    ]
  }
  ```

* **Wallet Not Found**

  ```json
  {
    "errors": [
      { "message": "Wallet does not exist" }
    ]
  }
  ```

* **Invalid Amount**

  ```json
  {
    "errors": [
      { "message": "Amount must be greater than zero" }
    ]
  }
  ```
---

