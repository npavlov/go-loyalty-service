package handlers

import (
	"net/http"

	"github.com/rs/zerolog"

	"github.com/npavlov/go-loyalty-service/internal/dbmanager"
)

type HandlerHealth struct {
	logger   *zerolog.Logger
	database *dbmanager.DBManager
}

// NewHealthHandler - constructor for HealthHandler.
func NewHealthHandler(database *dbmanager.DBManager, l *zerolog.Logger) *HandlerHealth {
	return &HandlerHealth{
		logger:   l,
		database: database,
	}
}

func (mh *HandlerHealth) Ping(response http.ResponseWriter, req *http.Request) {
	if !mh.database.IsConnected {
		mh.logger.Info().Msg("Database is not connected, can't ping")
		http.Error(response, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		return
	}

	if err := mh.database.DB.Ping(req.Context()); err != nil {
		mh.logger.Error().Err(err).Msg("No connection to database")
		http.Error(response, "Failed to connect to database: "+err.Error(), http.StatusInternalServerError)

		return
	}

	response.WriteHeader(http.StatusOK)
}
