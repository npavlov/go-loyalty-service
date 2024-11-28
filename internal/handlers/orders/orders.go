package handlers

import (
	"context"
	"io"
	"net/http"

	"github.com/npavlov/go-loyalty-service/internal/orders"
	"github.com/npavlov/go-loyalty-service/internal/storage"
	"github.com/npavlov/go-loyalty-service/internal/utils"
	"github.com/rs/zerolog"
)

type HandlerOrders struct {
	logger         *zerolog.Logger
	storage        *storage.DBStorage
	orderProcessor *orders.Orders
}

// NewOrdersHandler - constructor for HealthHandler.
func NewOrdersHandler(storage *storage.DBStorage, orderProc *orders.Orders, l *zerolog.Logger) *HandlerOrders {
	return &HandlerOrders{
		logger:         l,
		storage:        storage,
		orderProcessor: orderProc,
	}
}

func (mh *HandlerOrders) Get(response http.ResponseWriter, req *http.Request) {

	response.WriteHeader(http.StatusOK)
}

func (mh *HandlerOrders) Create(response http.ResponseWriter, req *http.Request) {
	// Read the request body
	body, err := io.ReadAll(req.Body)
	if err != nil {
		mh.logger.Error().Err(err).Msg("Order Create: error reading body")

		http.Error(response, "Unable to read request body", http.StatusBadRequest)

		return
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(req.Body) // Ensure the body is close

	orderNum := string(body)

	valid := utils.LuhnCheck(orderNum)

	if !valid {
		mh.logger.Error().Msg("Invalid order number")

		http.Error(response, "Invalid order number", http.StatusUnprocessableEntity)

		return
	}

	order, err := mh.storage.GetOrder(req.Context(), orderNum)
	if err != nil {
		mh.logger.Error().Err(err).Msg("Order Create: error getting order")

		return
	}

	currentUser := req.Context().Value("userID").(string)

	if order != nil && order.UserId.String() == currentUser {
		response.WriteHeader(http.StatusOK)

		return
	}

	if order != nil && order.UserId.String() != currentUser {
		response.WriteHeader(http.StatusConflict)

		return
	}

	newOrderId, err := mh.storage.CreateOrder(req.Context(), orderNum, currentUser)
	if err != nil {
		mh.logger.Error().Err(err).Msg("Order Create: error creating order")

		return
	}

	mh.logger.Info().Str("orderNum", orderNum).Str("Id", newOrderId).Msg("OrderId created")
	err = mh.orderProcessor.AddOrder(context.Background(), orderNum, currentUser)
	if err != nil {
		mh.logger.Err(err).Msg("Error adding order")
	}

	_, _ = response.Write([]byte(newOrderId))
	response.WriteHeader(http.StatusAccepted)
}
