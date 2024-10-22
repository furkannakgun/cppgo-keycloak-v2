package handler

import (
	"cpp/config"
	"cpp/db"
	"cpp/util"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

type Handler struct {
	DB          *gorm.DB
	OAuthConfig *oauth2.Config
	Config      *config.Config
}

func NewHandler(db *gorm.DB, oauthConfig *oauth2.Config, cfg *config.Config) *Handler {
	return &Handler{
		DB:          db,
		OAuthConfig: oauthConfig,
		Config:      cfg,
	}
}

func (h *Handler) LoginHandler(c *gin.Context) {
	loginUrl := h.OAuthConfig.AuthCodeURL("state")
	c.Redirect(http.StatusFound, loginUrl)
}

func (h *Handler) CallbackHandler(c *gin.Context) {
	code := c.Query("code")
	token, err := h.OAuthConfig.Exchange(c, code)
	if err != nil {
		log.Printf("Failed to exchange token: %v", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:    "jwt",
		Value:   token.AccessToken,
		Expires: time.Now().Add(24 * time.Hour),
		Path:    "/",
	})
	idToken, ok := token.Extra("id_token").(string)
	if ok {
		http.SetCookie(c.Writer, &http.Cookie{
			Name:    "id_token",
			Value:   idToken,
			Expires: time.Now().Add(24 * time.Hour),
			Path:    "/",
		})
	}
	c.Redirect(http.StatusFound, "/list")
}

func (h *Handler) LogoutHandler(c *gin.Context) {
	idToken, err := c.Cookie("id_token")
	if err != nil {
		log.Printf("Failed to retrieve id token: %v", err)
		c.Redirect(http.StatusFound, "/login")
		return
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "jwt",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
	})
	logoutURL := fmt.Sprintf("%s/auth/realms/%s/protocol/openid-connect/logout?id_token_hint=%s&post_logout_redirect_uri=%s",
		h.Config.KeycloakURL, h.Config.KeycloakRealm, url.QueryEscape(idToken),
		url.QueryEscape(fmt.Sprintf("%s/login", h.Config.CppHost)),
	)
	c.Redirect(http.StatusFound, logoutURL)
}

func (h *Handler) ListHandler(c *gin.Context) {
	query := c.DefaultQuery("query", "")
	var phoneNumbers []db.PhoneNumber
	if query != "" {
		h.DB.Where("phone_number LIKE ? OR display_name LIKE ?", "%"+query+"%", "%"+query+"%").Find(&phoneNumbers)
	} else {
		h.DB.Find(&phoneNumbers)
	}
	c.HTML(200, "list.html", gin.H{"data": phoneNumbers})
}

func (h *Handler) AddHandler(c *gin.Context) {
	var phoneNumber db.PhoneNumber
	if err := c.ShouldBind(&phoneNumber); err != nil {
		log.Printf("Error binding phone number: %v", err)
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	h.DB.Create(&phoneNumber)
	c.Redirect(302, "/")
}

func (h *Handler) IndexHandler(c *gin.Context) {
	c.HTML(200, "add.html", nil)
}

func (h *Handler) EditHandler(c *gin.Context) {
	id := c.Param("id")
	var phoneNumber db.PhoneNumber
	if err := h.DB.First(&phoneNumber, id).Error; err != nil {
		log.Printf("Error fetching phone number for editing: %v", err)
		c.JSON(404, gin.H{"error": "Phone number not found"})
		return
	}
	c.HTML(200, "edit.html", phoneNumber)
}

func (h *Handler) UpdateHandler(c *gin.Context) {
	id := c.Param("id")
	var phoneNumber db.PhoneNumber
	if err := h.DB.First(&phoneNumber, id).Error; err != nil {
		log.Printf("Error fetching phone number for updating: %v", err)
		c.JSON(404, gin.H{"error": "Phone number not found"})
		return
	}

	var updatedPhoneData db.PhoneNumber
	if err := c.ShouldBind(&updatedPhoneData); err != nil {
		log.Printf("Error binding updated phone number: %v", err)
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if updatedPhoneData.DisplayName != "" {
		phoneNumber.DisplayName = updatedPhoneData.DisplayName
	}
	log.Printf("Received form data: %+v", updatedPhoneData)

	if err := h.DB.Model(&phoneNumber).Updates(updatedPhoneData).Error; err != nil {
		log.Printf("Error saving updated phone number: %v", err)
		c.JSON(500, gin.H{"error": "Failed to update phone number"})
		return
	}
	c.Redirect(302, "/list")
}

func (h *Handler) DeleteHandler(c *gin.Context) {
	id := c.Param("id")
	var phoneNumber db.PhoneNumber
	if err := h.DB.First(&phoneNumber, id).Error; err != nil {
		log.Printf("Error fetching phone number for deletion: %v", err)
		c.JSON(404, gin.H{"error": "Phone number not found"})
		return
	}
	h.DB.Delete(&phoneNumber)
	c.Redirect(302, "/list")
}

func (h *Handler) LastHourCallsHandler(c *gin.Context) {
	phoneNumber := c.Param("phone_number")
	var count int64
	oneHourAgo := time.Now().Add(-1 * time.Hour)
	h.DB.Model(&db.CallLog{}).Joins("JOIN phone_numbers on phone_numbers.id = call_logs.phone_number_id").Where("phone_numbers.phone_number = ? AND call_logs.timestamp >= ?", phoneNumber, oneHourAgo).Count(&count)
	c.JSON(200, gin.H{"count": count})
}

func (h *Handler) MonitorHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "monitor.html", gin.H{"grafanaURL": h.Config.GrafanaURL})
}

func (h *Handler) CallLogsHandler(c *gin.Context) {
	phoneNumber := c.Query("phone_number")
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")
	sizeStr := c.DefaultQuery("size", "100")
	filter := c.DefaultQuery("filter", "")

	var results []struct {
		PhoneNumber       string
		DisplayName       string
		CalledPhoneNumber string
		Timestamp         time.Time
	}

	if filter == "" {
		if err := h.DB.Model(&db.CallLog{}).
			Joins("JOIN phone_numbers ON phone_numbers.id = call_logs.phone_number_id").
			Select("phone_numbers.phone_number, phone_numbers.display_name, call_logs.called_phone_number, call_logs.timestamp").
			Order("call_logs.timestamp desc").Limit(100).Find(&results).Error; err != nil {

			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve call logs"})
			return
		}

		formattedCallLogs := make([]FormattedCallLog, len(results))
		for i, result := range results {
			formattedCallLogs[i] = FormattedCallLog{
				PhoneNumber:        result.PhoneNumber,
				DisplayName:        result.DisplayName,
				CalledPhoneNumber:  result.CalledPhoneNumber,
				FormattedTimestamp: result.Timestamp.Format("02/01/2006 15:04:05"),
			}
		}
		c.HTML(http.StatusOK, "call_logs.html", gin.H{
			"CallLogs": formattedCallLogs,
		})
		return
	}

	size, err := strconv.Atoi(sizeStr)
	if err != nil || size < 1 || size > 1000 {
		size = 100
	}

	query := h.DB.Model(&db.CallLog{}).Limit(size)

	query = query.Joins("JOIN phone_numbers ON phone_numbers.id = call_logs.phone_number_id").
		Select("phone_numbers.phone_number, phone_numbers.display_name, call_logs.called_phone_number, call_logs.timestamp")

	if phoneNumber != "" {
		query = query.Where("phone_numbers.phone_number = ?", phoneNumber)
	}

	layout := "02-01-2006"
	var startDate, endDate time.Time
	var err1, err2 error

	if startDateStr != "" {
		startDate, err1 = time.Parse(layout, startDateStr)
		if err1 != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start date format"})
			return
		}
	} else {
		startDate = time.Now().AddDate(-5, 0, 0)
	}

	if endDateStr != "" {
		endDate, err2 = time.Parse(layout, endDateStr)
		if err2 != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end date format"})
			return
		}
	} else {
		endDate = time.Now().AddDate(0, 0, 1)
	}
	endDate = endDate.Add(24*time.Hour - 1)

	query = query.Where("call_logs.timestamp >= ? AND call_logs.timestamp <= ?", startDate, endDate)
	query = query.Order("call_logs.timestamp desc")

	if err := query.Find(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve call logs"})
		return
	}

	formattedCallLogs := make([]FormattedCallLog, len(results))
	for i, result := range results {
		formattedCallLogs[i] = FormattedCallLog{
			PhoneNumber:        result.PhoneNumber,
			DisplayName:        result.DisplayName,
			CalledPhoneNumber:  result.CalledPhoneNumber,
			FormattedTimestamp: result.Timestamp.Format("02/01/2006 15:04:05"),
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": formattedCallLogs})
}

func (h *Handler) CallNotificationHandler(c *gin.Context) {
	response, statusCode := h.mockCPPResponse(c, c.Param("network_id"), c.Param("service_id"))
	c.Writer.Header().Set("Content-Type", "application/json")
	c.JSON(statusCode, response)
}

func (h *Handler) mockCPPResponse(c *gin.Context, networkID string, serviceID string) (map[string]interface{}, int) {
	var request map[string]interface{}

	if err := c.ShouldBindJSON(&request); err != nil {
		return util.CustomErrorResponse(400, "Bad Request", "Error binding JSON")
	}

	if !util.IsValidNetworkAndService(networkID, serviceID) {
		return util.CustomErrorResponse(400, "Bad Request", "Invalid network or service ID")
	}

	callEventNotification, ok := request["callEventNotification"].(map[string]interface{})
	if !ok {
		return util.CustomErrorResponse(400, "Bad Request", "Invalid 'callEventNotification' structure")
	}

	callEvent, ok := callEventNotification["eventDescription"].(map[string]interface{})["callEvent"].(string)
	if !ok || callEvent == "" {
		return util.CustomErrorResponse(400, "Bad Request", "'callEvent' is missing or invalid")
	}

	callingParticipant, ok := callEventNotification["callingParticipant"].(string)
	if !ok || callingParticipant == "" {
		return util.CustomErrorResponse(400, "Bad Request", "Missing 'callingParticipant'")
	}

	calledParticipant, ok := callEventNotification["calledParticipant"].(string)
	if !ok || calledParticipant == "" {
		return util.CustomErrorResponse(400, "Bad Request", "Missing 'calledParticipant'")
	}

	var phoneNumber db.PhoneNumber
	if err := h.DB.Where("phone_number = ?", callingParticipant).First(&phoneNumber).Error; err != nil {
		return util.CustomErrorResponse(404, "Not Found", "Phone number not found")
	}

	logEntry := db.CallLog{PhoneNumberID: phoneNumber.ID, CalledPhoneNumber: calledParticipant}
	if err := h.DB.Create(&logEntry).Error; err != nil {
		return util.Custom502Error()
	}

	return map[string]interface{}{
		"action": map[string]interface{}{
			"actionToPerform": "Continue",
			"displayName":     phoneNumber.DisplayName,
		},
	}, 200
}
