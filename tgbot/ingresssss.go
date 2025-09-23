package main

import (
	"crypto/rand"
	"math/big"
	"time"
	_ "time/tzdata"
)

func PP(n int64) bool {
	if n > 100 {
		return true
	}
	if n <= 0 {
		return false
	}
	randInt, _ := rand.Int(rand.Reader, big.NewInt(int64(1000)))
	return randInt.Cmp(big.NewInt(n*10)) <= 0
}

func ingressStr() string {
	now := time.Now()
	if now.Weekday() == time.Saturday && now.Day() <= 7 {
		// IFS day
		if PP(27) {
			results := []string{
				"参加IFS",
				"女装参加IFS",
				"带猫猫参加IFS",
				"带狗狗参加IFS",
				"不带手机参加IFS",
				"在IFS现场转生",
				"刷AP",
				"女装刷AP",
				"嗑APEX刷AP",
				"请假刷AP",
				"翘班刷AP",
				"不带手机刷AP",
			}
			randInt, _ := rand.Int(rand.Reader, big.NewInt(int64(len(results))))
			return results[randInt.Int64()]
		}
	}
	if now.Weekday() == time.Sunday && now.Day() > 7 && now.Day() <= 14 {
		// ISS day
		if PP(17) {
			results := []string{
				"出门做一个ISS任务",
				"出门做两个ISS任务",
				"出门做三个ISS任务",
				"出门做四个ISS任务",
				"女装出门做ISS任务",
				"不带手机做ISS任务",
			}
			randInt, _ := rand.Int(rand.Reader, big.NewInt(int64(len(results))))
			return results[randInt.Int64()]
		}
	}
	return ""
}
