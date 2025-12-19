package model

import (
	"math/rand"
	"time"
)

// Questions содержит статичный набор вопросов для интервью.
var Questions = []string{
	"Объясните разницу между L1 и L2 регуляризацией.",
	"Как работает кросс-валидация k-fold?",
	"Что такое bias-variance tradeoff?",
	"Объясните принцип работы градиентного спуска.",
	"В чём разница между batch, mini-batch и stochastic gradient descent?",
	"Что такое overfitting и как с ним бороться?",
	"Объясните разницу между precision и recall.",
	"Что такое ROC-AUC и как его интерпретировать?",
	"Как работает метод опорных векторов (SVM)?",
	"Объясните принцип работы random forest.",
	"Что такое feature engineering и почему это важно?",
	"Как работает метод главных компонент (PCA)?",
	"Объясните разницу между supervised и unsupervised learning.",
	"Что такое regularization и зачем она нужна?",
	"Как работает алгоритм k-means кластеризации?",
	"Объясните концепцию ensemble методов.",
	"Что такое gradient boosting и как он работает?",
	"Как обрабатывать категориальные признаки?",
	"Что такое нормализация и стандартизация данных?",
	"Объясните принцип работы нейронных сетей.",
}

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

// GetRandomQuestion возвращает случайный вопрос из набора.
func GetRandomQuestion() string {
	return Questions[rng.Intn(len(Questions))]
}

// GetQuestionByIndex возвращает вопрос по индексу (с циклическим перебором).
func GetQuestionByIndex(index int) string {
	return Questions[index%len(Questions)]
}
