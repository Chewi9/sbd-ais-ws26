package rest

import (
	"net/http"
	"ordersystem/model" // to use model.Order in PostOrder
	"ordersystem/repository"

	"encoding/json" // read and write JSON

	"github.com/go-chi/render"
)

// GetMenu 			godoc
// @tags 			Menu
// @Description 	Returns the menu of all drinks
// @Produce  		json
// @Success 		200 {array} model.Drink
// @Router 			/api/menu [get]
func GetMenu(db *repository.DatabaseHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// todo
		// get slice from db
		drinks := db.GetDrinks() // we get the drinks slice

		render.Status(r, http.StatusOK)
		render.JSON(w, r, drinks)

	}
}

// todo create GetOrders /api/order/all
// GetOrders 			godoc
// @tags 			Order
// @Description 	Returns all orders
// @Produce  		json
// @Success 		200 {array} model.Order
// @Router 			/api/order/all [get]
func GetOrders(db *repository.DatabaseHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orders := db.GetOrders()

		render.Status(r, http.StatusOK)
		render.JSON(w, r, orders)
	}
}

// todo create GetOrdersTotal /api/order/total
// GetOrdersTotal 			godoc
// @tags 			Order
// @Description 	Returns the total of all orders
// @Produce  		json
// @Success 		200 {array} model.Order
// @Router 			/api/order/total [get]
func GetOrdersTotal(db *repository.DatabaseHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		totalledOrders := db.GetTotalledOrders()

		render.Status(r, http.StatusOK)
		render.JSON(w, r, totalledOrders)
	}
}

// PostOrder 		godoc
// @tags 			Order
// @Description 	Adds an order to the db
// @Accept 			json
// @Param 			b body model.Order true "Order"
// @Produce  		json
// @Success 		200
// @Failure     	400
// @Router 			/api/order [post]
func PostOrder(db *repository.DatabaseHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// todo
		// declare empty order struct

		var order model.Order // create order struct for the json

		decoder := json.NewDecoder(r.Body) // lector able to interpret json from the body
		err := decoder.Decode(&order) // need to modify the original, not a copy
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, "Invalid JSON")
			return
		}
		// handle error and render Status 400
		// add to db

		db.AddOrder(&order) // we pass the address of the order struct to AddOrder (AddOrder takes pointer to order struct, we do the reverse thing)

		render.Status(r, http.StatusOK)
		render.JSON(w, r, "ok")
	}
}
