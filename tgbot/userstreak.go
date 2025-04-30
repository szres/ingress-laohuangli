package main

import "time"

type streak struct {
	Date   string `json:"date"`
	Streak int    `json:"streak"`
}

func (lhl *laohuangli) getStreak(user int64) (now int, last int) {
	today := time.Now().Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	if _, ok := lhl.userStreak[user]; !ok {
		lhl.userStreak[user] = streak{Date: today, Streak: 1}
	}

	last = lhl.userStreak[user].Streak

	if lhl.userStreak[user].Date == today {
		now = lhl.userStreak[user].Streak
	}
	if lhl.userStreak[user].Date == yesterday {
		now = lhl.userStreak[user].Streak + 1
		lhl.userStreak[user] = streak{Date: today, Streak: now}
	}
	lhl.db.Write("datas", "streak", lhl.userStreak)
	return
}
