package test

import (
	"context"
	"errors"
	"log/slog"
	"math/rand"
	"net/http"
	"srmt-admin/internal/lib/model/test"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/storage"
)

type TestGetter interface {
	GetRandomGidroTest(ctx context.Context) (*test.GidroTest, error)
}

type Response struct {
	Question         string   `json:"question"`
	Answers          []string `json:"answers"`
	RightAnswerIndex int      `json:"right_answer_index"`
}

// New создает новый HTTP-хендлер для получения случайного теста.
func New(log *slog.Logger, getter TestGetter) http.HandlerFunc {
	// Создаем один источник случайных чисел для всего хендлера
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.telegram.gidro.test.New"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		// 1. Получаем случайный тест из БД
		test, err := getter.GetRandomGidroTest(r.Context())
		if err != nil {
			if errors.Is(err, storage.ErrDataNotFound) {
				log.Warn("no tests found in the database")
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("No tests found"))
				return
			}
			log.Error("failed to get random test", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve test"))
			return
		}

		// 2. Перемешиваем массив ответов (слайс структур)
		shuffledAnswers := test.Answers
		rnd.Shuffle(len(shuffledAnswers), func(i, j int) {
			shuffledAnswers[i], shuffledAnswers[j] = shuffledAnswers[j], shuffledAnswers[i]
		})

		// 3. Находим новый индекс правильного ответа и формируем срез строк
		newRightIndex := -1
		answerTexts := make([]string, len(shuffledAnswers))

		for i, answer := range shuffledAnswers {
			answerTexts[i] = answer.Text // Собираем тексты ответов для ответа клиенту
			if answer.IsCorrect {
				newRightIndex = i // Нашли новый индекс правильного ответа
			}
		}

		if newRightIndex == -1 {
			// Эта ситуация не должна произойти, если данные в БД консистентны
			log.Error("CRITICAL: right answer not found in answers list after shuffling", "question", test.QuestionText)
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Data consistency error"))
			return
		}

		// 4. Формируем и отправляем ответ
		response := Response{
			Question:         test.QuestionText,
			Answers:          answerTexts,
			RightAnswerIndex: newRightIndex,
		}

		log.Info("successfully served a random test")
		render.JSON(w, r, response)
	}
}
