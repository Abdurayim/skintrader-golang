package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"skintrader-go/internal/domain"
)

type GameHandler struct {
	gameRepo domain.GameRepository
	logger   zerolog.Logger
}

func NewGameHandler(gameRepo domain.GameRepository, logger zerolog.Logger) *GameHandler {
	return &GameHandler{
		gameRepo: gameRepo,
		logger:   logger.With().Str("handler", "game").Logger(),
	}
}

func (h *GameHandler) GetGames(c *gin.Context) {
	games, err := h.gameRepo.FindActive(c.Request.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get games")
		Error(c, err)
		return
	}
	Success(c, gin.H{"games": games}, "Games retrieved")
}

func (h *GameHandler) SearchGames(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		BadRequest(c, "Search query is required")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	games, total, err := h.gameRepo.SearchByName(c.Request.Context(), query, limit, offset)
	if err != nil {
		h.logger.Error().Err(err).Str("query", query).Msg("failed to search games")
		Error(c, err)
		return
	}

	Paginated(c, gin.H{"games": games}, gin.H{"total": total, "limit": limit, "offset": offset}, "Games found")
}

func (h *GameHandler) GetPopularGames(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	games, err := h.gameRepo.GetPopular(c.Request.Context(), limit)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get popular games")
		Error(c, err)
		return
	}
	Success(c, gin.H{"games": games}, "Popular games retrieved")
}

func (h *GameHandler) GetGenres(c *gin.Context) {
	Success(c, gin.H{"genres": domain.GameGenres}, "Genres retrieved")
}

func (h *GameHandler) GetGamesByGenre(c *gin.Context) {
	genre := c.Param("genre")
	if genre == "" {
		BadRequest(c, "Genre is required")
		return
	}

	games, err := h.gameRepo.FindByGenre(c.Request.Context(), genre)
	if err != nil {
		h.logger.Error().Err(err).Str("genre", genre).Msg("failed to get games by genre")
		Error(c, err)
		return
	}
	Success(c, gin.H{"games": games}, "Games retrieved")
}

func (h *GameHandler) GetGame(c *gin.Context) {
	identifier := c.Param("identifier")

	if id, err := uuid.Parse(identifier); err == nil {
		game, err := h.gameRepo.FindByID(c.Request.Context(), id)
		if err != nil {
			Error(c, err)
			return
		}
		Success(c, gin.H{"game": game}, "Game retrieved")
		return
	}

	game, err := h.gameRepo.FindBySlug(c.Request.Context(), identifier)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, gin.H{"game": game}, "Game retrieved")
}
