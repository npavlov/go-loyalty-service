package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/rs/zerolog"

	"github.com/npavlov/go-loyalty-service/internal/orders"
	"github.com/npavlov/go-loyalty-service/internal/storage"
	"github.com/npavlov/go-loyalty-service/internal/utils"
)

type HandlerOrders struct {
	logger         *zerolog.Logger
	storage        storage.Storage
	orderProcessor orders.QueueProcessor
}

// NewOrdersHandler - constructor for HealthHandler.
func NewOrdersHandler(storage storage.Storage, orderProc orders.QueueProcessor, l *zerolog.Logger) *HandlerOrders {
	return &HandlerOrders{
		logger:         l,
		storage:        storage,
		orderProcessor: orderProc,
	}
}

func (mh *HandlerOrders) GetOrders(response http.ResponseWriter, req *http.Request) {
	currentUser := req.Context().Value("userID").(string)

	dbOrders, err := mh.storage.GetOrders(req.Context(), currentUser)
	if err != nil {
		mh.logger.Error().Err(err).Msg("error getting orders")

		http.Error(response, "error getting orders", http.StatusInternalServerError)

		return
	}

	if len(dbOrders) == 0 {
		mh.logger.Info().Msg("no orders found")

		http.Error(response, "no orders found", http.StatusNoContent)

		return
	}

	responseData, err := json.Marshal(dbOrders)
	if err != nil {
		mh.logger.Error().Err(err).Msg("failed to marshal response")
		http.Error(response, "internal server error", http.StatusInternalServerError)

		return
	}

	response.WriteHeader(http.StatusOK)
	_, _ = response.Write(responseData)
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

	currentUser := req.Context().Value("userID").(string)

	order, err := mh.storage.GetOrder(req.Context(), orderNum)
	if err != nil {
		mh.logger.Error().Err(err).Msg("Order Create: error getting order")

		return
	}

	if order != nil && order.UserId.String() == currentUser {
		mh.logger.Info().Str("orderNum", orderNum).Msg("Order is already created by this user")
		response.WriteHeader(http.StatusOK)

		return
	}

	if order != nil && order.UserId.String() != currentUser {
		mh.logger.Info().Str("orderNum", orderNum).Msg("Order is already created by other user")
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

	response.WriteHeader(http.StatusAccepted)
	_, _ = response.Write([]byte(newOrderId))
}
