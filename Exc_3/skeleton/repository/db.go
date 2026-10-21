package repository

import "ordersystem/model"
import "time"

type DatabaseHandler struct {
	// drinks represent all available drinks
	drinks []model.Drink
	// orders serves as order history
	orders []model.Order
}

// todo
func NewDatabaseHandler() *DatabaseHandler {
	// Init the drinks slice with some test data
	drinks := []model.Drink{
		{ID: 1, Name: "Cola", Description: "Classic Cola", Price: 2.00},
		{ID: 2, Name: "Fanta", Description: "Orange Fanta", Price: 2.00},
		{ID: 3, Name: "Water", Description: "Mineral Water", Price: 1.50},
	}

	// Init orders slice with some test data
	orders := []model.Order{
		{DrinkID: 1, CreatedAt: time.Now(), Amount: 2},
		{DrinkID: 2, CreatedAt: time.Now(), Amount: 1},
		{DrinkID: 3, CreatedAt: time.Now(), Amount: 3},

	}

	return &DatabaseHandler{
		drinks: drinks,
		orders: orders,
	}
}

func (db *DatabaseHandler) GetDrinks() []model.Drink {
	return db.drinks
}

func (db *DatabaseHandler) GetOrders() []model.Order {
	return db.orders
}

// todo
func (db *DatabaseHandler) GetTotalledOrders() map[uint64]uint64 {

	totalledOrders := make(map[uint64]uint64) // each drink ID will be a key, and the value will be the total amount of orders for that drink
	for _, order := range db.orders { // iterate over all orders
		totalledOrders[order.DrinkID] += uint64(order.Amount) // accumulate the amount of orders for each DrinkID
	}

	// calculate total orders
	// key = DrinkID, value = Amount of orders
	// totalledOrders map[uint64]uint64
	return totalledOrders
}

func (db *DatabaseHandler) AddOrder(order *model.Order) {
	// todo
	// add order to db.orders slice
	db.orders = append(db.orders, *order) // We add the new order to orders slice, we dont lose the original order data
	// db.orders is the slice of orders in the database
	// order is the new order to be added
	// *order is the value of the order

}
