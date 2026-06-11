package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/gin-gonic/gin"
)

type PushService struct {
	db         *sql.DB
	publicKey  string
	privateKey string
	subject    string
}

type pushPayload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url"`
	Type  string `json:"type"`
}

type pushSubscriptionInput struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

type pushPreferencesInput struct {
	UserID            int                   `json:"userId"`
	Subscription      pushSubscriptionInput `json:"subscription"`
	Endpoint          string                `json:"endpoint"`
	NotifyLeaderboard bool                  `json:"notifyLeaderboard"`
	NotifyMatchStart  bool                  `json:"notifyMatchStart"`
}

type unsubscribeInput struct {
	Endpoint string `json:"endpoint"`
}

func NewPushService(database *sql.DB) *PushService {
	return &PushService{
		db:         database,
		publicKey:  os.Getenv("VAPID_PUBLIC_KEY"),
		privateKey: os.Getenv("VAPID_PRIVATE_KEY"),
		subject:    os.Getenv("VAPID_SUBJECT"),
	}
}

func (p *PushService) PublicKey() string {
	return p.publicKey
}

func GetVAPIDPublicKey(push *PushService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if push.PublicKey() == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "VAPID_PUBLIC_KEY not set"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"publicKey": push.PublicKey()})
	}
}

func SubscribePush(database *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input pushPreferencesInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if input.Subscription.Endpoint == "" || input.Subscription.Keys.P256dh == "" || input.Subscription.Keys.Auth == "" || input.UserID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "userId and subscription endpoint/keys are required"})
			return
		}

		now := time.Now().UTC().Format(time.RFC3339)
		_, err := database.Exec(`
			INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth, notify_leaderboard, notify_match_start, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(endpoint) DO UPDATE SET
				user_id = excluded.user_id,
				p256dh = excluded.p256dh,
				auth = excluded.auth,
				notify_leaderboard = excluded.notify_leaderboard,
				notify_match_start = excluded.notify_match_start,
				updated_at = excluded.updated_at
		`, input.UserID, input.Subscription.Endpoint, input.Subscription.Keys.P256dh, input.Subscription.Keys.Auth, input.NotifyLeaderboard, input.NotifyMatchStart, now, now)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "subscribed"})
	}
}

func UpdatePushPreferences(database *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input pushPreferencesInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		endpoint := input.Endpoint
		if endpoint == "" {
			endpoint = input.Subscription.Endpoint
		}
		if endpoint == "" || input.UserID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "endpoint and userId are required"})
			return
		}

		result, err := database.Exec(`
			UPDATE push_subscriptions
			SET user_id = ?, notify_leaderboard = ?, notify_match_start = ?, updated_at = ?
			WHERE endpoint = ?
		`, input.UserID, input.NotifyLeaderboard, input.NotifyMatchStart, time.Now().UTC().Format(time.RFC3339), endpoint)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if affected, _ := result.RowsAffected(); affected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "updated"})
	}
}

func UnsubscribePush(database *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input unsubscribeInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if input.Endpoint == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "endpoint is required"})
			return
		}

		_, err := database.Exec("DELETE FROM push_subscriptions WHERE endpoint = ?", input.Endpoint)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "unsubscribed"})
	}
}

func (p *PushService) NotifyMatchStart(matchID string, homeTeamID, awayTeamID int64, homeTeam, awayTeam string) {
	p.sendToSubscriptions(`
		SELECT ps.id, ps.endpoint, ps.p256dh, ps.auth
		FROM push_subscriptions ps
		JOIN user_teams ut ON ut.user_id = ps.user_id
		WHERE ps.notify_match_start = 1 AND ut.team_id IN (?, ?)
		GROUP BY ps.id
	`, []any{homeTeamID, awayTeamID}, "match-start:"+matchID, pushPayload{
		Title: "Match starting",
		Body:  homeTeam + " vs " + awayTeam + " is live.",
		URL:   "/",
		Type:  "match-start",
	})
}

func (p *PushService) NotifyLeaderboardChange(userID, oldRank, newRank, points int) {
	eventKey := "leaderboard-rank:" + itoa(userID) + ":" + itoa(oldRank) + ":" + itoa(newRank) + ":" + itoa(points)
	p.sendToSubscriptions(`
		SELECT id, endpoint, p256dh, auth
		FROM push_subscriptions
		WHERE notify_leaderboard = 1 AND user_id = ?
	`, []any{userID}, eventKey, pushPayload{
		Title: "New leaderboard position",
		Body:  "You moved from " + ordinal(oldRank) + " to " + ordinal(newRank) + ".",
		URL:   "/leaderboard",
		Type:  "leaderboard",
	})
}

func (p *PushService) sendToSubscriptions(query string, args []any, eventKey string, payload pushPayload) {
	if p.publicKey == "" || p.privateKey == "" || p.subject == "" {
		return
	}

	rows, err := p.db.Query(query, args...)
	if err != nil {
		return
	}
	defer rows.Close()

	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	for rows.Next() {
		var id int64
		var endpoint, p256dh, auth string
		if err := rows.Scan(&id, &endpoint, &p256dh, &auth); err != nil {
			continue
		}
		if p.deliveryExists(id, eventKey) {
			continue
		}

		resp, err := webpush.SendNotification(data, &webpush.Subscription{
			Endpoint: endpoint,
			Keys: webpush.Keys{
				P256dh: p256dh,
				Auth:   auth,
			},
		}, &webpush.Options{
			Subscriber:      p.subject,
			VAPIDPublicKey:  p.publicKey,
			VAPIDPrivateKey: p.privateKey,
			TTL:             30,
		})
		if resp != nil {
			resp.Body.Close()
		}
		if isExpiredPushResponse(resp) {
			p.deleteSubscription(id)
			continue
		}
		if resp != nil && (resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices) {
			continue
		}
		if err != nil {
			continue
		}
		p.recordDelivery(id, eventKey)
	}
}

func (p *PushService) deliveryExists(subscriptionID int64, eventKey string) bool {
	var exists int
	err := p.db.QueryRow("SELECT 1 FROM notification_deliveries WHERE subscription_id = ? AND event_key = ?", subscriptionID, eventKey).Scan(&exists)
	return err == nil
}

func (p *PushService) recordDelivery(subscriptionID int64, eventKey string) {
	_, _ = p.db.Exec("INSERT OR IGNORE INTO notification_deliveries (subscription_id, event_key, sent_at) VALUES (?, ?, ?)", subscriptionID, eventKey, time.Now().UTC().Format(time.RFC3339))
}

func (p *PushService) deleteSubscription(subscriptionID int64) {
	_, _ = p.db.Exec("DELETE FROM push_subscriptions WHERE id = ?", subscriptionID)
}

func isExpiredPushResponse(resp *http.Response) bool {
	return resp != nil && (resp.StatusCode == http.StatusGone || resp.StatusCode == http.StatusNotFound)
}

func ordinal(n int) string {
	suffix := "th"
	if n%100 < 11 || n%100 > 13 {
		switch n % 10 {
		case 1:
			suffix = "st"
		case 2:
			suffix = "nd"
		case 3:
			suffix = "rd"
		}
	}
	return itoa(n) + suffix
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
