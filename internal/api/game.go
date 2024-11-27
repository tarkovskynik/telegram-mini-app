package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"UD_telegram_miniapp/internal/repository"
	"UD_telegram_miniapp/internal/service"
	"UD_telegram_miniapp/pkg/auth"
	"UD_telegram_miniapp/pkg/logger"
	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type ballGameRoutes struct {
	repo service.UserRepository
	a    *auth.TelegramAuth
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Game struct {
	PlayerID        int64
	TotalEnergy     int
	RemainingEnergy int
	IsPlaying       bool
	IsReadyToPlay   bool
	HitCounter      int
	TotalScore      int
	CurrentHitScore int
	conn            *websocket.Conn
	mu              sync.Mutex
}

type Message struct {
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload,omitempty"`
}

var (
	activeGames = make(map[int64]*Game)
	gamesMutex  sync.RWMutex
)

var (
	basePointReward  = 10
	bonusPointReward = 0
)

func NewGameRoutes(handler *gin.RouterGroup, repo service.UserRepository, a *auth.TelegramAuth) {
	r := &ballGameRoutes{repo: repo, a: a}
	h := handler.Group("/ws")

	h.GET("/:telegram_id", r.handleWebSocket)

	//admin
	admin := handler.Group("/admin")
	admin.DELETE("/:telegram_id/reset-energy", r.ResetEnergy)
}

func (gr *ballGameRoutes) handleWebSocket(c *gin.Context) {
	uID := c.Param("telegram_id")
	userID, err := strconv.ParseInt(uID, 10, 64)
	if err != nil {
		log.Println(fmt.Errorf("invalid telegram_id: %s", uID))
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("websocket upgrade failed", zap.Error(err))
		return
	}

	game := &Game{
		PlayerID:      userID,
		IsPlaying:     false,
		IsReadyToPlay: false,
		conn:          conn,
	}

	gamesMutex.Lock()
	activeGames[userID] = game
	gamesMutex.Unlock()

	go gr.handleGameLoop(game)
}

func (gr *ballGameRoutes) handleGameLoop(game *Game) {
	defer func() {
		game.conn.Close()
		gamesMutex.Lock()
		delete(activeGames, game.PlayerID)
		gamesMutex.Unlock()
	}()

	for {
		_, msg, err := game.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Println("websocket unexpected close", zap.Error(err))
			}
			break
		}

		var message Message
		if err := json.Unmarshal(msg, &message); err != nil {
			log.Println("failed to unmarshal message", zap.Error(err))
			continue
		}

		total, remaining, err := gr.repo.GetPlayerEnergy(context.TODO(), game.PlayerID)
		if err != nil {
			log.Println("failed to get player energy")
			return
		}

		if !game.IsPlaying {
			game.TotalEnergy = total
			game.RemainingEnergy = remaining

			if game.RemainingEnergy <= 0 {
				game.IsReadyToPlay = false
			} else {
				game.IsReadyToPlay = true
			}
		}

		switch message.Type {
		case "player_state":
			gr.sendPlayerState(game)

		case "game_start":
			if !game.IsReadyToPlay {
				_, usedAt, err := gr.repo.GetEnergyStatus(context.TODO(), game.PlayerID)
				if err != nil {
					log.Printf("failed to get player energy: %s", err.Error())
					return
				}

				nextAvailableEnergy := usedAt.UTC().Add(8 * time.Hour).Unix()
				gr.sendError(game, "out of energy", nextAvailableEnergy)

				continue
			}

			if !game.IsPlaying {
				game.IsPlaying = true
				game.HitCounter = 0
				game.TotalScore = 0
				game.CurrentHitScore = 0
				bonusPointReward = 0
				game.RemainingEnergy--
				gr.sendGameState(game)
			}

		case "ball_hit":
			if !game.IsReadyToPlay {
				_, usedAt, err := gr.repo.GetEnergyStatus(context.TODO(), game.PlayerID)
				if err != nil {
					log.Printf("failed to get player energy: %s", err)
					return
				}

				nextAvailableEnergy := usedAt.UTC().Add(8 * time.Hour).Unix()
				gr.sendError(game, "out of energy", nextAvailableEnergy)

				continue
			}

			if game.IsPlaying {
				game.CurrentHitScore = basePointReward + bonusPointReward
				bonusPointReward++
				game.TotalScore = game.TotalScore + game.CurrentHitScore
				game.HitCounter++
				gr.sendGameState(game)
			}

		case "ball_dropped":
			if !game.IsReadyToPlay {
				_, usedAt, err := gr.repo.GetEnergyStatus(context.TODO(), game.PlayerID)
				if err != nil {
					log.Println("failed to get player energy")
					return
				}

				nextAvailableEnergy := usedAt.UTC().Add(8 * time.Hour).Unix()
				gr.sendError(game, "out of energy", nextAvailableEnergy)
				continue
			}

			if game.IsPlaying {
				game.IsPlaying = false
				gr.handleGameOver(game)
			}

		case "energy_recharge":
			out := Message{
				Type: "energy_recharge",
				Payload: map[string]interface{}{
					"energy_recharge_status": "success",
				},
			}

			outJson, err := json.MarshalIndent(out, "", "	")
			if err != nil {
				log.Println("failed to marshal json", zap.Error(err))
			}

			err = game.conn.WriteMessage(websocket.TextMessage, outJson)
			if err != nil {
				log.Println("failed to write message", zap.Error(err))
			}
		}
	}
}

func (gr *ballGameRoutes) sendPlayerState(game *Game) {
	l := logger.Logger()

	state := Message{
		Type: "player_state",
		Payload: map[string]any{
			"total_energy":     game.TotalEnergy,
			"remaining_energy": game.RemainingEnergy,
		},
	}

	data, err := json.MarshalIndent(state, "", "	")
	if err != nil {
		l.Error("failed to marshal player state",
			zap.Int64("player_id", game.PlayerID),
			zap.Error(err))
		return
	}

	if err := game.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		l.Error("failed to send player state",
			zap.Int64("player_id", game.PlayerID),
			zap.Error(err))
	}
}

func (gr *ballGameRoutes) sendGameState(game *Game) {
	state := Message{
		Type: "game_state",
		Payload: map[string]any{
			"total_score":       game.TotalScore,
			"current_hit_score": game.CurrentHitScore,
			"hit_counter":       game.HitCounter,
			"total_energy":      game.TotalEnergy,
			"remaining_energy":  game.RemainingEnergy,
			"is_playing":        game.IsPlaying,
		},
	}

	data, err := json.MarshalIndent(state, "", "	")
	if err != nil {
		log.Println("failed to marshal game state", zap.Error(err))
		return
	}

	if err := game.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Println("failed to send game state", zap.Error(err))
	}
}

func (gr *ballGameRoutes) sendError(game *Game, message string, nextAvailableEnergy int64) {
	m := Message{
		Type: "error",
		Payload: map[string]any{
			"message":                    message,
			"next_available_energy_unix": nextAvailableEnergy,
		},
	}

	data, err := json.MarshalIndent(m, "", "	")
	if err != nil {
		log.Println("failed to marshal game state", zap.Error(err))
		return
	}

	if err := game.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Println("failed to send game state", zap.Error(err))
	}
}

func (gr *ballGameRoutes) handleGameOver(game *Game) {
	game.IsPlaying = false

	gameOver := Message{
		Type: "game_over",
		Payload: map[string]any{
			"final_score":       game.TotalScore,
			"final_hit_counter": game.HitCounter,
			"remaining_energy":  game.RemainingEnergy,
			"is_playing":        game.IsPlaying,
		},
	}

	data, err := json.MarshalIndent(gameOver, "", "	")
	if err != nil {
		log.Println("failed to marshal game over message", zap.Error(err))
		return
	}

	if err := game.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Println("failed to send game over message", zap.Error(err))
	}

	if err := gr.repo.UpdateUserPoints(context.Background(), game.PlayerID, game.TotalScore); err != nil {
		log.Println("failed to update user points",
			zap.Int64("player_id", game.PlayerID),
			zap.Error(err))
	}

	if err := gr.repo.UpdatePlayerEnergy(context.Background(), game.PlayerID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			log.Println("player not found", zap.Int64("player_id", game.PlayerID))
			return
		}
		log.Println("failed to update energy",
			zap.Int64("player_id", game.PlayerID),
			zap.Error(err))
	}
}

func (gr *ballGameRoutes) ResetEnergy(c *gin.Context) {
	log := logger.Logger()

	telegramID := c.Param("telegram_id")
	id, err := strconv.ParseInt(telegramID, 10, 64)
	if err != nil {
		log.Error("failed to parse telegram_id", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid telegram_id"})
		return
	}

	err = gr.repo.ResetEnergy(c.Request.Context(), id)
	if err != nil {
		log.Error("failed to reset energy",
			zap.Int64("telegram_id", id),
			zap.Error(err))

		if errors.Is(err, service.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reset energy"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "energy reset successful",
		"telegram_id": id,
	})
}
