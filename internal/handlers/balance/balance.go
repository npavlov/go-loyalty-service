package balance

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog"

	"github.com/npavlov/go-loyalty-service/internal/models"
	"github.com/npavlov/go-loyalty-service/internal/storage"
	"github.com/npavlov/go-loyalty-service/internal/utils"
)

type HandlerBalance struct {
	logger  *zerolog.Logger
	storage *storage.DBStorage
}

// NewBalanceHandler - constructor for HealthHandler.
func NewBalanceHandler(storage *storage.DBStorage, l *zerolog.Logger) *HandlerBalance {
	return &HandlerBalance{
		logger:  l,
		storage: storage,
	}
}

func (mh *HandlerBalance) GetBalance(response http.ResponseWriter, req *http.Request) {
	currentUser := req.Context().Value("userID").(string)

	dbBalance, err := mh.storage.GetBalance(req.Context(), currentUser)
	if err != nil {
		mh.logger.Error().Err(err).Msg("error getting orders")

		http.Error(response, "error getting orders", http.StatusInternalServerError)

		return
	}

	responseData, err := json.Marshal(dbBalance)
	if err != nil {
		mh.logger.Error().Err(err).Msg("failed to marshal response")
		http.Error(response, "internal server error", http.StatusInternalServerError)

		return
	}

	// Write response
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(http.StatusOK)
	_, _ = response.Write(responseData)
}

func (mh *HandlerBalance) MakeWithdrawal(response http.ResponseWriter, req *http.Request) {
	// Read the request body
	var mkWithdrawal models.MakeWithdrawal
	if err := json.NewDecoder(req.Body).Decode(&mkWithdrawal); err != nil {
		http.Error(response, "Invalid request", http.StatusBadRequest)
		return
	}

	valid := utils.LuhnCheck(mkWithdrawal.Order)

	if !valid {
		mh.logger.Error().Msg("Invalid order number")

		http.Error(response, "Invalid order number", http.StatusUnprocessableEntity)

		return
	}

	currentUser := req.Context().Value("userID").(string)

	balance, err := mh.storage.GetBalance(req.Context(), currentUser)
	if err != nil {
		mh.logger.Error().Err(err).Msg("Withdrawal Create: error getting balance")

		http.Error(response, "Withdrawal Create: error getting balance", http.StatusInternalServerError)

		return
	}

	if balance.Balance < mkWithdrawal.Sum {
		mh.logger.Error().Msg("Withdrawal Create: insufficient balance")

		http.Error(response, "Withdrawal Create: insufficient balance", http.StatusPaymentRequired)

		return
	}

	err = mh.storage.MakeWithdrawn(req.Context(), currentUser, mkWithdrawal.Order, mkWithdrawal.Sum)
	if err != nil {
		mh.logger.Error().Err(err).Msg("Order Create: error adding withdrawal")

		http.Error(response, "error adding withdrawal", http.StatusInternalServerError)

		return
	}

	response.WriteHeader(http.StatusOK)
}

// GetWithdrawals handles the `GET /api/user/withdrawals` endpoint.
func (mh *HandlerBalance) GetWithdrawals(response http.ResponseWriter, req *http.Request) {
	// Retrieve the authenticated user ID from the context
	currentUser := req.Context().Value("userID").(string)

	// Fetch withdrawals from storage
	withdrawals, err := mh.storage.GetWithdrawals(req.Context(), currentUser)
	if err != nil {
		mh.logger.Error().Err(err).Msg("failed to fetch withdrawals")
		http.Error(response, "Internal Server Error", http.StatusInternalServerError)

		return
	}

	// If no withdrawals found, return 204 No Content
	if len(withdrawals) == 0 {
		response.WriteHeader(http.StatusNoContent)

		return
	}

	if err := json.NewEncoder(response).Encode(withdrawals); err != nil {
		mh.logger.Error().Err(err).Msg("failed to encode response")

		http.Error(response, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	response.WriteHeader(http.StatusOK)
}
