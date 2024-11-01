package service

import (
	"context"
	"testing"
	"time"

	"UD_telegram_miniapp/internal/model"
	"UD_telegram_miniapp/internal/repository"
	"UD_telegram_miniapp/internal/service/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDailyQuestService_GetStatus(t *testing.T) {
	mockRepo := &mocks.MockDailyQuestRepository{}
	service := NewDailyQuestService(mockRepo)

	tests := []struct {
		name            string
		telegramID      int64
		mockSetup       func()
		expectedQuest   *model.DailyQuest
		expectedError   error
		checkAdditional func(*testing.T, *model.DailyQuest)
	}{
		{
			name:       "User not found",
			telegramID: 123,
			mockSetup: func() {
				mockRepo.On("GetDailyQuestStatus", mock.Anything, int64(123)).
					Return(nil, repository.ErrNotFound)
			},
			expectedQuest: nil,
			expectedError: ErrUserNotFound,
		},
		{
			name:       "Never claimed before",
			telegramID: 124,
			mockSetup: func() {
				mockRepo.On("GetDailyQuestStatus", mock.Anything, int64(124)).
					Return(&model.DailyQuest{
						UserTelegramID:         124,
						LastClaimedAt:          nil,
						ConsecutiveDaysClaimed: 0,
					}, nil)
			},
			expectedQuest: &model.DailyQuest{
				UserTelegramID:         124,
				LastClaimedAt:          nil,
				ConsecutiveDaysClaimed: 0,
				HasNeverBeenClaimed:    true,
				IsAvailable:            true,
				NextClaimAvailable:     nil,
				DailyRewards:           make([]model.DayReward, 7),
			},
			checkAdditional: func(t *testing.T, quest *model.DailyQuest) {
				for i := 0; i < 7; i++ {
					assert.Equal(t, i+1, quest.DailyRewards[i].Day)
					assert.Equal(t, BaseReward+DailyBonuses[i], quest.DailyRewards[i].Reward)
				}
			},
		},
		{
			name:       "Recently claimed (not available)",
			telegramID: 125,
			mockSetup: func() {
				lastClaimed := time.Now().Add(-12 * time.Hour)
				mockRepo.On("GetDailyQuestStatus", mock.Anything, int64(125)).
					Return(&model.DailyQuest{
						UserTelegramID:         125,
						LastClaimedAt:          &lastClaimed,
						ConsecutiveDaysClaimed: 2,
					}, nil)
			},
			checkAdditional: func(t *testing.T, quest *model.DailyQuest) {
				assert.False(t, quest.IsAvailable)
				assert.NotNil(t, quest.NextClaimAvailable)
				assert.Equal(t, 2, quest.ConsecutiveDaysClaimed)
			},
		},
		{
			name:       "Available after 24 hours",
			telegramID: 126,
			mockSetup: func() {
				lastClaimed := time.Now().Add(-25 * time.Hour)
				mockRepo.On("GetDailyQuestStatus", mock.Anything, int64(126)).
					Return(&model.DailyQuest{
						UserTelegramID:         126,
						LastClaimedAt:          &lastClaimed,
						ConsecutiveDaysClaimed: 3,
					}, nil)
			},
			checkAdditional: func(t *testing.T, quest *model.DailyQuest) {
				assert.True(t, quest.IsAvailable)
				assert.NotNil(t, quest.NextClaimAvailable)
				assert.Equal(t, 3, quest.ConsecutiveDaysClaimed)
			},
		},
		{
			name:       "Expired (after 48 hours)",
			telegramID: 127,
			mockSetup: func() {
				lastClaimed := time.Now().Add(-49 * time.Hour)
				mockRepo.On("GetDailyQuestStatus", mock.Anything, int64(127)).
					Return(&model.DailyQuest{
						UserTelegramID:         127,
						LastClaimedAt:          &lastClaimed,
						ConsecutiveDaysClaimed: 4,
					}, nil)

				mockRepo.On("UpdateDailyQuestStatus", mock.Anything, mock.MatchedBy(func(quest *model.DailyQuest) bool {
					return quest.ConsecutiveDaysClaimed == 0
				})).Return(nil)
			},
			checkAdditional: func(t *testing.T, quest *model.DailyQuest) {
				assert.True(t, quest.IsAvailable)
				assert.NotNil(t, quest.NextClaimAvailable)
				assert.Equal(t, 0, quest.ConsecutiveDaysClaimed)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			quest, err := service.GetStatus(context.Background(), tt.telegramID)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedError)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, quest)

			if tt.checkAdditional != nil {
				tt.checkAdditional(t, quest)
			}
		})
	}

	mockRepo.AssertExpectations(t)
}

func TestDailyQuestService_Claim(t *testing.T) {
	mockRepo := &mocks.MockDailyQuestRepository{}
	service := NewDailyQuestService(mockRepo)

	tests := []struct {
		name           string
		telegramID     int64
		setupMocks     func(mockRepo *mocks.MockDailyQuestRepository)
		expectedError  error
		checkMockCalls func(t *testing.T, mockRepo *mocks.MockDailyQuestRepository)
	}{
		{
			name:       "Successful first claim",
			telegramID: 123,
			setupMocks: func(mockRepo *mocks.MockDailyQuestRepository) {
				mockRepo.On("GetDailyQuestStatus", mock.Anything, int64(123)).
					Return(&model.DailyQuest{
						UserTelegramID:         123,
						LastClaimedAt:          nil,
						ConsecutiveDaysClaimed: 0,
					}, nil)

				mockRepo.On("UpdateDailyQuestStatus", mock.Anything, mock.MatchedBy(func(quest *model.DailyQuest) bool {
					return quest.UserTelegramID == 123 &&
						quest.LastClaimedAt != nil &&
						quest.ConsecutiveDaysClaimed == 1
				})).Return(nil)

				mockRepo.On("UpdateUserPoints", mock.Anything, int64(123), BaseReward+DailyBonuses[0]).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name:       "Successful consecutive claim (day 3)",
			telegramID: 124,
			setupMocks: func(mockRepo *mocks.MockDailyQuestRepository) {
				lastClaimed := time.Now().Add(-25 * time.Hour)
				mockRepo.On("GetDailyQuestStatus", mock.Anything, int64(124)).
					Return(&model.DailyQuest{
						UserTelegramID:         124,
						LastClaimedAt:          &lastClaimed,
						ConsecutiveDaysClaimed: 2,
					}, nil)

				mockRepo.On("UpdateDailyQuestStatus", mock.Anything, mock.MatchedBy(func(quest *model.DailyQuest) bool {
					return quest.UserTelegramID == 124 &&
						quest.LastClaimedAt != nil &&
						quest.ConsecutiveDaysClaimed == 3
				})).Return(nil)

				mockRepo.On("UpdateUserPoints", mock.Anything, int64(124), BaseReward+DailyBonuses[2]).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name:       "Successful claim and verify next claim time",
			telegramID: 128,
			setupMocks: func(mockRepo *mocks.MockDailyQuestRepository) {
				mockRepo.On("GetDailyQuestStatus", mock.Anything, int64(128)).
					Return(&model.DailyQuest{
						UserTelegramID:         128,
						LastClaimedAt:          nil,
						ConsecutiveDaysClaimed: 0,
					}, nil).Once()

				testStart := time.Now()
				mockRepo.On("UpdateDailyQuestStatus", mock.Anything,
					mock.MatchedBy(func(quest *model.DailyQuest) bool {
						return quest.UserTelegramID == 128 &&
							quest.LastClaimedAt != nil &&
							time.Since(*quest.LastClaimedAt) < 2*time.Second &&
							quest.ConsecutiveDaysClaimed == 1
					})).Return(nil)

				mockRepo.On("UpdateUserPoints", mock.Anything, int64(128),
					BaseReward+DailyBonuses[0]).Return(nil)

				mockRepo.On("GetDailyQuestStatus", mock.Anything, int64(128)).
					Return(&model.DailyQuest{
						UserTelegramID:         128,
						LastClaimedAt:          &testStart,
						ConsecutiveDaysClaimed: 1,
					}, nil).Once()
			},
			expectedError: nil,
			checkMockCalls: func(t *testing.T, mockRepo *mocks.MockDailyQuestRepository) {
				status, err := service.GetStatus(context.Background(), 128)
				assert.NoError(t, err)
				assert.NotNil(t, status)

				assert.False(t, status.IsAvailable)
				assert.NotNil(t, status.NextClaimAvailable)
				assert.NotNil(t, status.LastClaimedAt)

				timeUntilNextClaim := status.NextClaimAvailable.Sub(*status.LastClaimedAt)
				assert.True(t, timeUntilNextClaim > 23*time.Hour)
				assert.True(t, timeUntilNextClaim <= 24*time.Hour)

				assert.Equal(t, 1, status.ConsecutiveDaysClaimed)
			},
		},
		{
			name:       "Reset after day 7",
			telegramID: 125,
			setupMocks: func(mockRepo *mocks.MockDailyQuestRepository) {
				lastClaimed := time.Now().Add(-25 * time.Hour)
				mockRepo.On("GetDailyQuestStatus", mock.Anything, int64(125)).
					Return(&model.DailyQuest{
						UserTelegramID:         125,
						LastClaimedAt:          &lastClaimed,
						ConsecutiveDaysClaimed: 7,
					}, nil)

				mockRepo.On("UpdateDailyQuestStatus", mock.Anything, mock.MatchedBy(func(quest *model.DailyQuest) bool {
					return quest.UserTelegramID == 125 &&
						quest.LastClaimedAt != nil &&
						quest.ConsecutiveDaysClaimed == 1
				})).Return(nil)

				mockRepo.On("UpdateUserPoints", mock.Anything, int64(125), BaseReward+DailyBonuses[0]).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name:       "Claim not available",
			telegramID: 126,
			setupMocks: func(mockRepo *mocks.MockDailyQuestRepository) {
				lastClaimed := time.Now().Add(-12 * time.Hour)
				mockRepo.On("GetDailyQuestStatus", mock.Anything, int64(126)).
					Return(&model.DailyQuest{
						UserTelegramID:         126,
						LastClaimedAt:          &lastClaimed,
						ConsecutiveDaysClaimed: 1,
					}, nil)
			},
			expectedError: ErrClaimNotAvailable,
		},
		{
			name:       "Update points error",
			telegramID: 127,
			setupMocks: func(mockRepo *mocks.MockDailyQuestRepository) {
				lastClaimed := time.Now().Add(-25 * time.Hour)
				mockRepo.On("GetDailyQuestStatus", mock.Anything, int64(127)).
					Return(&model.DailyQuest{
						UserTelegramID:         127,
						LastClaimedAt:          &lastClaimed,
						ConsecutiveDaysClaimed: 1,
					}, nil)

				mockRepo.On("UpdateDailyQuestStatus", mock.Anything, mock.Anything).
					Return(nil)

				mockRepo.On("UpdateUserPoints", mock.Anything, int64(127), mock.Anything).
					Return(assert.AnError)
			},
			expectedError: assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo.ExpectedCalls = nil
			mockRepo.Calls = nil

			tt.setupMocks(mockRepo)

			err := service.Claim(context.Background(), tt.telegramID)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			if tt.checkMockCalls != nil {
				tt.checkMockCalls(t, mockRepo)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}
