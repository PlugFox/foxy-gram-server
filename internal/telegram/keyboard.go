package telegram

import (
	tele "gopkg.in/telebot.v3"
)

const captchaKeyboardUnique = "captcha-keyboard"

type captchaKeyboard struct {
	unique   string
	numbers  []tele.InlineButton
	refresh  tele.InlineButton
	cancel   tele.InlineButton
	solve    tele.InlineButton
	keyboard [][]tele.InlineButton
}

func captchaKeyboardDefault() captchaKeyboard {
	numbers := make([]tele.InlineButton, 0, 10)
	numbers = append(numbers, tele.InlineButton{Text: "0️⃣", Unique: captchaKeyboardUnique, Data: "captcha-zero"})
	numbers = append(numbers, tele.InlineButton{Text: "1️⃣", Unique: captchaKeyboardUnique, Data: "captcha-one"})
	numbers = append(numbers, tele.InlineButton{Text: "2️⃣", Unique: captchaKeyboardUnique, Data: "captcha-two"})
	numbers = append(numbers, tele.InlineButton{Text: "3️⃣", Unique: captchaKeyboardUnique, Data: "captcha-three"})
	numbers = append(numbers, tele.InlineButton{Text: "4️⃣", Unique: captchaKeyboardUnique, Data: "captcha-four"})
	numbers = append(numbers, tele.InlineButton{Text: "5️⃣", Unique: captchaKeyboardUnique, Data: "captcha-five"})
	numbers = append(numbers, tele.InlineButton{Text: "6️⃣", Unique: captchaKeyboardUnique, Data: "captcha-six"})
	numbers = append(numbers, tele.InlineButton{Text: "7️⃣", Unique: captchaKeyboardUnique, Data: "captcha-seven"})
	numbers = append(numbers, tele.InlineButton{Text: "8️⃣", Unique: captchaKeyboardUnique, Data: "captcha-eight"})
	numbers = append(numbers, tele.InlineButton{Text: "9️⃣", Unique: captchaKeyboardUnique, Data: "captcha-nine"})
	refresh := tele.InlineButton{Text: "🔄", Unique: captchaKeyboardUnique, Data: "captcha-refresh"}
	solve := tele.InlineButton{Text: "↩️", Unique: captchaKeyboardUnique, Data: "captcha-backspace"}

	keyboard := [][]tele.InlineButton{
		{numbers[1], numbers[2], numbers[3]},
		{numbers[4], numbers[5], numbers[6]},
		{numbers[7], numbers[8], numbers[9]},
		{refresh, numbers[0], solve},
	}

	return captchaKeyboard{
		unique:   captchaKeyboardUnique,
		numbers:  numbers,
		refresh:  refresh,
		solve:    solve,
		keyboard: keyboard,
	}
}
