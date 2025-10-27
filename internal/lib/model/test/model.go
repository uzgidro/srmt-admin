package test

type Answer struct {
	Text      string `bson:"text"`
	IsCorrect bool   `bson:"is_correct"`
}

type GidroTest struct {
	QuestionText string   `bson:"question_text"`
	Answers      []Answer `bson:"answers"`
}
