package main

import "time"

type userstreak struct {
	Date   string `json:"date"`
	Streak int    `json:"streak"`
	All    int    `json:"all"`
	Weeks  int    `json:"weeks"`
}

func (lhl *laohuangli) updateStreak(user int64) {
	today := time.Now().Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	if _, ok := lhl.userStreak[user]; !ok {
		lhl.userStreak[user] = userstreak{Date: today, Streak: 1, All: 1, Weeks: 0}
		return
	}

	date := lhl.userStreak[user].Date
	streak := lhl.userStreak[user].Streak
	all := lhl.userStreak[user].All
	weeks := lhl.userStreak[user].Weeks

	// temp fix
	if streak > all {
		all = streak
	}
	if streak >= 7 {
		weeks = 1
	}
	// should be removed before 5.14
	// temp fix end

	if date == today {
		return
	}
	all++
	if date == yesterday {
		streak++
		if streak%7 == 0 {
			weeks++
		}
	} else {
		streak = 1
	}
	lhl.userStreak[user] = userstreak{Date: today, Streak: streak, All: all, Weeks: weeks}
	lhl.db.Write("datas", "streak", lhl.userStreak)
}
func (lhl *laohuangli) getStreak(user int64) (week int, day int) {
	day = lhl.userStreak[user].Streak
	week = lhl.userStreak[user].Weeks
	return
}
