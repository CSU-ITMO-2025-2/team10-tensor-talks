package model

// InterviewProgram представляет программу интервью.
type InterviewProgram struct {
	Questions []QuestionItem `json:"questions"`
}

// QuestionItem представляет один пункт программы (вопрос + теория).
type QuestionItem struct {
	Question string `json:"question"`
	Theory   string `json:"theory"`
	Order    int    `json:"order"`
}

// GetStaticInterviewProgram возвращает статичную программу интервью.
// В будущем здесь будет логика генерации программы на основе параметров.
func GetStaticInterviewProgram() InterviewProgram {
	return InterviewProgram{
		Questions: []QuestionItem{
			{
				Question: "Объясните разницу между L1 и L2 регуляризацией.",
				Theory:   "L1 регуляризация (Lasso) добавляет сумму абсолютных значений весов к функции потерь, что способствует обнулению некоторых весов. L2 регуляризация (Ridge) добавляет сумму квадратов весов, что уменьшает веса, но не обнуляет их.",
				Order:    1,
			},
			{
				Question: "Как работает кросс-валидация k-fold?",
				Theory:   "Кросс-валидация k-fold разделяет данные на k частей. Модель обучается на k-1 частях и проверяется на оставшейся части. Процесс повторяется k раз, каждый раз используя другую часть для валидации.",
				Order:    2,
			},
			{
				Question: "Что такое bias-variance tradeoff?",
				Theory:   "Bias-variance tradeoff - это баланс между ошибкой смещения (bias) и ошибкой дисперсии (variance). Высокий bias приводит к недообучению, высокий variance - к переобучению. Оптимальная модель балансирует эти два источника ошибок.",
				Order:    3,
			},
			{
				Question: "Объясните принцип работы градиентного спуска.",
				Theory:   "Градиентный спуск - это итеративный алгоритм оптимизации, который минимизирует функцию потерь, двигаясь в направлении отрицательного градиента. На каждом шаге веса обновляются пропорционально градиенту функции потерь.",
				Order:    4,
			},
			{
				Question: "В чём разница между batch, mini-batch и stochastic gradient descent?",
				Theory:   "Batch GD использует все данные для вычисления градиента на каждой итерации. Mini-batch GD использует небольшие подмножества данных. Stochastic GD использует один пример на итерацию. Mini-batch является компромиссом между скоростью и стабильностью.",
				Order:    5,
			},
		},
	}
}
