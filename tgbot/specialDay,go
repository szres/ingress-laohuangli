package main

import (
	"crypto/rand"
	"math/big"
	"time"
	_ "time/tzdata"
)

// TODO: use external data for specail events
func specialDay() string {
	now := time.Now()
	if now.Year() == 2025 && now.Month() == 9 && (now.Day() == 24 || now.Day() == 25) {
		if PP(31) {
			results := []string{
				"居家避风狂刷AP",
				"穿救生衣去hack",
				"发型被吹成鸟窝",
				"囤积泡面抗台风",
				"翘班躲风暴deploy",
				"搭橡皮艇上班",
				"投食避难邻居",
				"拒接台风预警电话",
				"迎风link",
				"变得湿透",
				"随风起飞",
				"拉field避风",
				"用冰箱门当救生艇",
				"用洗衣机冲浪",
				"用台风声冒充白噪音冥想",
				"囤菜最后只吃泡面",
				"用应急灯打光自拍刷AP",
				"台风眼里偷溜出去买菜",
				"把暴雨当免费洗车",
				"在积水的客厅划橡皮艇",
				"随风翻滚deploy",
				"用外卖箱当船link",
				"撑伞逆风安装resonator",
			}
			randInt, _ := rand.Int(rand.Reader, big.NewInt(int64(len(results))))
			return results[randInt.Int64()]
		}
	}
	return ""
}
