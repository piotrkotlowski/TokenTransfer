package main

import (
    "errors"
    "fmt"           
    "net/http"
    "strings"        
    "gorm.io/gorm"
    "gorm.io/gorm/clause"
    "github.com/graphql-go/graphql"
    "github.com/graphql-go/handler"
)


type Wallet struct {
	Address string  `gorm:"primaryKey"`
	Balance float64 `gorm:"not null"`
}

type Transaction struct {
	TransactionID uint    `gorm:"primaryKey;autoIncrement;column:transaction_id" json:"transaction_id"`
	Sender        string  `gorm:"index;not null"`
	Receiver      string  `gorm:"index;not null"`
	Amount        float64 `gorm:"not null"`
}


var walletType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Wallet",
	Fields: graphql.Fields{
		"address": &graphql.Field{Type: graphql.String},
		"balance": &graphql.Field{Type: graphql.Float},
	},
})

var txType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Transaction",
	Fields: graphql.Fields{
		"transaction_id": &graphql.Field{Type: graphql.Int},
		"sender":         &graphql.Field{Type: graphql.String},
		"receiver":       &graphql.Field{Type: graphql.String},
		"amount":         &graphql.Field{Type: graphql.Float},
	},
})
func NewGraphQLHandler(db *gorm.DB) http.Handler {

	/* Query */
	queryType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"wallets": &graphql.Field{
				Type: graphql.NewList(walletType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					var wallets []Wallet
					if err := db.Find(&wallets).Error; err != nil {
						return nil, err
					}
					return wallets, nil
				},
			},
			"transactions": &graphql.Field{
				Type: graphql.NewList(txType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					var txs []Transaction
					if err := db.Order("transaction_id DESC").Find(&txs).Error; err != nil {
						return nil, err
					}
					return txs, nil
				},
			},
		},
	})


	/* Mutation */
	mutationType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Mutation",
		Fields: graphql.Fields{
			"createWallet": &graphql.Field{
				Type: walletType,
				Args: graphql.FieldConfigArgument{
					"address": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
					"balance": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.Float),
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					addr := p.Args["address"].(string)
					bal  := p.Args["balance"].(float64)
	
					
					var existing Wallet
					if err := db.
						
						First(&existing, "address = ?", addr).Error; err == nil {
						return nil, fmt.Errorf("wallet %q already exists", addr)
					} else if !errors.Is(err, gorm.ErrRecordNotFound) {
						return nil, err 
					}
	

					w := Wallet{Address: addr, Balance: bal}
					if err := db.Create(&w).Error; err != nil {
						if strings.Contains(err.Error(), "duplicate") ||
							strings.Contains(err.Error(), "unique") {
							return nil, fmt.Errorf("wallet %q already exists", addr)
						}
						return nil, err
					}
					return w, nil
				},
			},
			"makeTransaction": &graphql.Field{
				Type: txType,
				Args: graphql.FieldConfigArgument{
					"sender":   &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
					"receiver": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
					"amount":   &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.Float)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					sender   := p.Args["sender"].(string)
					receiver := p.Args["receiver"].(string)
					amount   := p.Args["amount"].(float64)
			
					
					if amount <= 0 {
						return nil, errors.New("amount must be positive")
					}
			
					var resultTx Transaction
					err := db.Transaction(func(tx *gorm.DB) error {
						
						var senderWallet Wallet
						if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
							First(&senderWallet, "address = ?", sender).Error; err != nil {
							if errors.Is(err, gorm.ErrRecordNotFound) {
								return errors.New("wallet not in database") 
							}
							return err
						}
						if senderWallet.Balance < amount {
							return errors.New("insufficient amount")
						}
			
						
						var receiverWallet Wallet
						if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
							First(&receiverWallet, "address = ?", receiver).Error; err != nil {
							if errors.Is(err, gorm.ErrRecordNotFound) {
								return errors.New("wallet not in database") // receiver missing
							}
							return err
						}
			
						
						if err := tx.Model(&Wallet{}).
							Where("address = ?", sender).
							Update("balance", gorm.Expr("balance - ?", amount)).Error; err != nil {
							return err
						}
						if err := tx.Model(&Wallet{}).
							Where("address = ?", receiver).
							Update("balance", gorm.Expr("balance + ?", amount)).Error; err != nil {
							return err
						}
			
						
						resultTx = Transaction{
							Sender:   sender,
							Receiver: receiver,
							Amount:   amount,
						}
						return tx.Create(&resultTx).Error
					})
					if err != nil {
						return nil, err
					}
					return resultTx, nil
				},
			},
		},
	})

	schema, _ := graphql.NewSchema(graphql.SchemaConfig{
		Query:    queryType,
		Mutation: mutationType,
	})

	return handler.New(&handler.Config{
		Schema:   &schema,
		Pretty:   true,
		GraphiQL: true,
	})
}