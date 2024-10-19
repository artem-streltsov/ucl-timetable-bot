package handlers

import (
	"fmt"
	"log"
	"strings"

	"github.com/artem-streltsov/ucl-timetable-bot/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) handleAddFriend(user *models.User, text string) {
	if !strings.HasPrefix(text, "@") || len(text) < 2 {
		h.sendMessage(user.ChatID, "Invalid username format. Please provide a valid Telegram username (e.g., @username).")
		return
	}

	friendUsername := strings.TrimPrefix(text, "@")
	friend, err := h.db.GetUserByUsername(friendUsername)
	if err != nil {
		h.sendMessage(user.ChatID, "Error accessing the database. Please try again later.")
		return
	}
	if friend == nil {
		h.sendMessage(user.ChatID, "User not found. Please ensure the user has started the bot and their username is correct.")
		return
	}
	if friend.ChatID == user.ChatID {
		h.sendMessage(user.ChatID, "You cannot add yourself as a friend.")
		return
	}

	areFriends, err := h.db.AreFriends(user.ChatID, friend.ChatID)
	if err != nil {
		h.sendMessage(user.ChatID, "Error checking friendship status.")
		return
	}
	if areFriends {
		h.sendMessage(user.ChatID, "You are already friends with this user.")
		return
	}

	requestExists, err := h.db.FriendRequestExists(user.ChatID, friend.ChatID)
	if err != nil {
		h.sendMessage(user.ChatID, "Error checking existing friend requests.")
		return
	}
	if requestExists {
		h.sendMessage(user.ChatID, "You have already sent a friend request to this user.")
		return
	}

	err = h.db.AddFriendRequest(user.ChatID, friend.ChatID)
	if err != nil {
		h.sendMessage(user.ChatID, "Error sending friend request.")
		return
	}

	h.sendMessage(user.ChatID, "Request sent. Your friend now needs to add you with the /accept_friend command.")
	h.clearUserState(user.ChatID)

	h.sendMessage(friend.ChatID, fmt.Sprintf("@%s has sent you a friend request. Use /accept_friend to accept.", user.Username))
}

func (h *Handler) handleAcceptFriend(user *models.User) {
	requestorIDs, err := h.db.GetPendingFriendRequests(user.ChatID)
	if err != nil {
		h.sendMessage(user.ChatID, "Error fetching friend requests.")
		return
	}

	if len(requestorIDs) == 0 {
		h.sendMessage(user.ChatID, "You have no pending friend requests.")
		h.clearUserState(user.ChatID)
		return
	}

	var buttons [][]tgbotapi.InlineKeyboardButton
	for _, requestorID := range requestorIDs {
		requestor, err := h.db.GetUser(requestorID)
		if err != nil || requestor == nil {
			continue
		}
		requestorUsername := requestor.Username

		callbackData := fmt.Sprintf("accept_%d", requestorID)
		button := tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("@%s", requestorUsername), callbackData)
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(button))
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
	msg := tgbotapi.NewMessage(user.ChatID, "Pending Friend Requests:")
	msg.ReplyMarkup = keyboard

	if _, err := h.api.Send(msg); err != nil {
		log.Printf("Error sending friend requests: %v", err)
	}

	h.clearUserState(user.ChatID)
}
