package scheduler

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var (
	ErrEmptyRepeat   = errors.New("repeat: пустая строка")
	ErrBadStartDate  = errors.New("dstart: некорректная дата")
	ErrBadRepeat     = errors.New("repeat: неверный формат")
	ErrUnsupported   = errors.New("repeat: неподдерживаемый формат")
	ErrIntervalRange = errors.New("repeat d: интервал вне допустимого диапазона 1..400")
)

// NextDate вычисляет следующую дату исполнения задачи согласно правилу повторения.
// now    — текущая дата/время, относительно которого дата результата должна быть строго больше
// dstart — исходная дата (YYYYMMDD), от которой начинается отсчёт (всегда делаем хотя бы один шаг вперёд)
// repeat — правило повторения: "d <N>", "y", "w <days>", "m <days> [months]"
func NextDate(now time.Time, dstart string, repeat string) (string, error) {
	repeat = strings.TrimSpace(repeat)
	if repeat == "" {
		return "", ErrEmptyRepeat
	}
	start, err := time.Parse("20060102", strings.TrimSpace(dstart))
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrBadStartDate, err)
	}
	// Нормализуем сравнение по датам (обнуляем время)
	now = dateOnly(now)
	start = dateOnly(start)

	parts := strings.Fields(repeat)
	if len(parts) == 0 {
		return "", ErrBadRepeat
	}
	switch strings.ToLower(parts[0]) {
	case "y":
		if len(parts) != 1 {
			return "", ErrBadRepeat
		}
		next := start
		// Делать минимум один шаг вперёд от dstart
		next = addOneYearKeepOverflow(next)
		for !afterNow(next, now) {
			next = addOneYearKeepOverflow(next)
		}
		return next.Format("20060102"), nil

	case "d":
		if len(parts) != 2 {
			return "", ErrBadRepeat
		}
		interval, err := strconv.Atoi(parts[1])
		if err != nil || interval < 1 || interval > 400 {
			return "", ErrIntervalRange
		}
		next := start
		// Минимум один шаг
		next = next.AddDate(0, 0, interval)
		for !afterNow(next, now) {
			next = next.AddDate(0, 0, interval)
		}
		return next.Format("20060102"), nil

	case "w":
		// Формат: w <через запятую из 1..7>
		// Пример: w 1,4,5
		if len(parts) != 2 {
			return "", ErrBadRepeat
		}
		weekdays, err := parseWeekdays(parts[1])
		if err != nil {
			return "", err
		}
		next := nextWeeklyAfter(start, weekdays)
		for !afterNow(next, now) {
			next = nextWeeklyAfter(next, weekdays)
		}
		return next.Format("20060102"), nil

	case "m":
		// Формат: m <дни> [месяцы]
		// дни: 1..31, -1, -2
		// месяцы (опционально): 1..12
		if len(parts) != 2 && len(parts) != 3 {
			return "", ErrBadRepeat
		}
		monthDays, err := parseMonthDays(parts[1])
		if err != nil {
			return "", err
		}
		var monthsAllowed map[int]struct{}
		if len(parts) == 3 {
			monthsAllowed, err = parseMonths(parts[2])
			if err != nil {
				return "", err
			}
		}
		next := nextMonthlyAfter(start, monthDays, monthsAllowed)
		for !afterNow(next, now) {
			next = nextMonthlyAfter(next, monthDays, monthsAllowed)
		}
		return next.Format("20060102"), nil
	default:
		return "", ErrUnsupported
	}
}

// afterNow возвращает true, если d строго больше now по дате (время игнорируется).
func afterNow(d, now time.Time) bool {
	d = dateOnly(d)
	now = dateOnly(now)
	return d.After(now)
}

func dateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.Local)
}

// addOneYearKeepOverflow добавляет 1 год, сохраняя "переполнение" дня в следующий месяц,
// чтобы 29 февраля корректно превращался в 1 марта невисокосного года.
func addOneYearKeepOverflow(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y+1, m, d, 0, 0, 0, 0, t.Location())
}

// parseWeekdays парсит список "1,4,5" в множество допустимых дней недели (1=Пн..7=Вс)
func parseWeekdays(s string) (map[int]struct{}, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, ErrBadRepeat
	}
	out := make(map[int]struct{})
	for _, tok := range strings.Split(s, ",") {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			return nil, ErrBadRepeat
		}
		n, err := strconv.Atoi(tok)
		if err != nil || n < 1 || n > 7 {
			return nil, fmt.Errorf("repeat w: недопустимый день недели %q", tok)
		}
		out[n] = struct{}{}
	}
	if len(out) == 0 {
		return nil, ErrBadRepeat
	}
	return out, nil
}

// parseMonthDays парсит список дней месяца "1,15,25" с поддержкой -1 (последний), -2 (предпоследний)
func parseMonthDays(s string) ([]int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, ErrBadRepeat
	}
	seen := make(map[int]struct{})
	var out []int
	for _, tok := range strings.Split(s, ",") {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			return nil, ErrBadRepeat
		}
		n, err := strconv.Atoi(tok)
		if err != nil {
			return nil, ErrBadRepeat
		}
		if !((n >= 1 && n <= 31) || n == -1 || n == -2) {
			return nil, fmt.Errorf("repeat m: недопустимый день месяца %q", tok)
		}
		if _, ok := seen[n]; !ok {
			seen[n] = struct{}{}
			out = append(out, n)
		}
	}
	if len(out) == 0 {
		return nil, ErrBadRepeat
	}
	return out, nil
}

// parseMonths парсит список месяцев "1,3,6" (1..12)
func parseMonths(s string) (map[int]struct{}, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, ErrBadRepeat
	}
	out := make(map[int]struct{})
	for _, tok := range strings.Split(s, ",") {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			return nil, ErrBadRepeat
		}
		n, err := strconv.Atoi(tok)
		if err != nil || n < 1 || n > 12 {
			return nil, fmt.Errorf("repeat m: недопустимый месяц %q", tok)
		}
		out[n] = struct{}{}
	}
	if len(out) == 0 {
		return nil, ErrBadRepeat
	}
	return out, nil
}

// nextWeeklyAfter возвращает ближайшую дату строго после t, попадающую в набор будних дней.
func nextWeeklyAfter(t time.Time, weekdays map[int]struct{}) time.Time {
	cur := dateOnly(t).AddDate(0, 0, 1) // начинаем со следующего дня, т.к. "строго после"
	for i := 0; i < 8; i++ {
		if _, ok := weekdays[weekdayISO(cur)]; ok {
			return cur
		}
		cur = cur.AddDate(0, 0, 1)
	}
	// Теоретически недостижимо, но вернём cur
	return cur
}

// weekdayISO возвращает номер дня недели: 1=Пн .. 7=Вс.
func weekdayISO(t time.Time) int {
	w := int(t.Weekday()) // 0=Вс, 1=Пн .. 6=Сб
	if w == 0 {
		return 7
	}
	return w
}

// nextMonthlyAfter возвращает ближайшую дату строго после t, подходящую по правилам дней месяца и (опц.) месяцев.
func nextMonthlyAfter(t time.Time, monthDays []int, monthsAllowed map[int]struct{}) time.Time {
	cur := dateOnly(t)
	y, m, _ := cur.Date()
	loc := cur.Location()

	// Начинаем проверку с текущего месяца
	for i := 0; i < 2400; i++ { // ограничение защиты от бесконечного цикла (~200 лет вперёд максимум)
		if monthsAllowed == nil || isMonthAllowed(int(m), monthsAllowed) {
			// сформировать кандидатов в этом месяце
			candidates := monthlyCandidates(y, m, loc, monthDays)
			// выбрать минимальную дату строго больше cur
			var best time.Time
			for _, c := range candidates {
				if c.After(cur) && (best.IsZero() || c.Before(best)) {
					best = c
				}
			}
			if !best.IsZero() {
				return best
			}
		}
		// переход к следующему месяцу
		nextMonth := time.Date(y, m, 1, 0, 0, 0, 0, loc).AddDate(0, 1, 0)
		y, m, _ = nextMonth.Date()
		// Сдвинем опорную дату на последний день прошлого месяца, чтобы "строго после"
		cur = time.Date(y, m, 0, 0, 0, 0, 0, loc)
	}
	// fallback: очень далеко в будущем
	return cur.AddDate(100, 0, 0)
}

func isMonthAllowed(mon int, allowed map[int]struct{}) bool {
	_, ok := allowed[mon]
	return ok
}

func monthlyCandidates(year int, month time.Month, loc *time.Location, monthDays []int) []time.Time {
	last := lastDayOfMonth(year, month, loc)
	var out []time.Time
	for _, d := range monthDays {
		switch {
		case d >= 1 && d <= 31:
			if d <= last.Day() {
				out = append(out, time.Date(year, month, d, 0, 0, 0, 0, loc))
			}
		case d == -1:
			out = append(out, last)
		case d == -2:
			out = append(out, last.AddDate(0, 0, -1))
		default:
			// уже провалидировано
		}
	}
	return out
}

func lastDayOfMonth(year int, month time.Month, loc *time.Location) time.Time {
	// День 0 следующего месяца — это последний день текущего месяца
	return time.Date(year, month+1, 0, 0, 0, 0, 0, loc)
}


