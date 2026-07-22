package model

func InitSeedData() error {
	seeders := []func() error{
		seedPackages,
		seedNodeAreas,
		seedNodes,
		seedAdverts,
		seedDelayedPopups,
	}
	for _, seeder := range seeders {
		if err := seeder(); err != nil {
			return err
		}
	}
	return nil
}

func seedDelayedPopups() error {
	var count int64
	if err := DB.Model(&DelayedPopup{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	return DB.Create(&DelayedPopup{
		Title:     "服务迁移提醒",
		Content:   "如果当前软件长时间无法连接，请使用转移码前往新软件兑换会员权益",
		ImageUrl:  "https://example.com/images/transfer-popup.png",
		CanClose:  1,
		DelayDays: 3,
		Status:    DelayedPopupStatusEnabled,
		Platforms: "all",
		Versions:  "all",
		Sorter:    100,
	}).Error
}

func seedPackages() error {
	var count int64
	if err := DB.Model(&Package{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	packages := []Package{
		{
			AppleId:     "just_vpn_vip_month",
			Name:        "月度会员",
			SubName:     "连续30天高速线路",
			Selected:    1,
			Corner:      "推荐",
			Value:       "畅享全部会员线路",
			Price:       1990,
			OriginPrice: "¥29.9",
			PakTips:     "¥19.9",
			Remark:      "适合短期使用",
			Day:         30,
			Status:      1,
			Sorter:      10,
		},
		{
			AppleId:     "just_vpn_vip_quarter",
			Name:        "季度会员",
			SubName:     "连续90天高速线路",
			Selected:    0,
			Corner:      "省心",
			Value:       "适合稳定使用",
			Price:       4990,
			OriginPrice: "¥89.9",
			PakTips:     "¥49.9",
			Remark:      "平均每天不到1元",
			Day:         90,
			Status:      1,
			Sorter:      20,
		},
		{
			AppleId:     "just_vpn_vip_year",
			Name:        "年度会员",
			SubName:     "连续365天高速线路",
			Selected:    0,
			Corner:      "超值",
			Value:       "适合长期使用",
			Price:       15900,
			OriginPrice: "¥299",
			PakTips:     "¥159",
			Remark:      "全年高速线路可用",
			Day:         365,
			Status:      1,
			Sorter:      30,
		},
	}
	return DB.Create(&packages).Error
}

func seedNodeAreas() error {
	var count int64
	if err := DB.Model(&NodeArea{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	areas := []NodeArea{
		{
			Code:        "FREE",
			Name:        "免费线路",
			Sort:        1,
			MinConnTime: 80,
			MaxConnTime: 180,
			Status:      1,
			ImgUrl:      "https://example.com/images/line-free.png",
		},
		{
			Code:        "HK",
			Name:        "香港",
			Sort:        10,
			MinConnTime: 60,
			MaxConnTime: 160,
			Status:      1,
			ImgUrl:      "https://example.com/images/line-hk.png",
		},
		{
			Code:        "US",
			Name:        "美国",
			Sort:        20,
			MinConnTime: 120,
			MaxConnTime: 260,
			Status:      1,
			ImgUrl:      "https://example.com/images/line-us.png",
		},
	}
	return DB.Create(&areas).Error
}

func seedNodes() error {
	var count int64
	if err := DB.Model(&Node{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	nodes := []Node{
		{
			Code:     "FREE",
			CodeName: "免费线路",
			Name:     "免费体验节点 1",
			NodeType: "vless",
			LinkUrl:  "vless://free-test-node@example.com:443?security=tls&type=ws#FREE-Test-1",
			Address:  "free-test-node.example.com",
			Status:   NodeStatusEnabled,
		},
		{
			Code:     "HK",
			CodeName: "香港",
			Name:     "香港高速节点 1",
			NodeType: "vless",
			LinkUrl:  "vless://hk-test-node@example.com:443?security=tls&type=ws#HK-Test-1",
			Address:  "hk-test-node.example.com",
			Status:   NodeStatusEnabled,
		},
		{
			Code:     "US",
			CodeName: "美国",
			Name:     "美国高速节点 1",
			NodeType: "vless",
			LinkUrl:  "vless://us-test-node@example.com:443?security=tls&type=ws#US-Test-1",
			Address:  "us-test-node.example.com",
			Status:   NodeStatusEnabled,
		},
		{
			Code:     "AUTO",
			CodeName: "智能线路",
			Name:     "智能兜底节点 1",
			NodeType: "vless",
			LinkUrl:  "vless://auto-test-node@example.com:443?security=tls&type=ws#AUTO-Test-1",
			Address:  "auto-test-node.example.com",
			Status:   NodeStatusEnabled,
		},
	}
	return DB.Create(&nodes).Error
}

func seedAdverts() error {
	var count int64
	if err := DB.Model(&Advert{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	adverts := []Advert{
		{
			Position:     AdvertPositionSplash,
			Title:        "启动页广告",
			Content:      "欢迎使用高速网络服务",
			ImageUrl:     "https://example.com/images/ad-splash.png",
			LinkUrl:      "https://www.baidu.com",
			MaxShowTimes: 1,
			Platforms:    "all",
			Versions:     "all",
			Sorter:       300,
		},
		{
			Position:     AdvertPositionHomePopup,
			Title:        "会员限时优惠",
			Content:      "开通会员后可使用全部高速线路",
			ImageUrl:     "https://example.com/images/ad-home-popup.png",
			LinkUrl:      "https://www.baidu.com",
			MaxShowTimes: 3,
			Platforms:    "all",
			Versions:     "all",
			Sorter:       200,
		},
		{
			Position:     AdvertPositionBanner,
			Title:        "首页 Banner",
			Content:      "高速线路已准备就绪",
			ImageUrl:     "https://example.com/images/ad-banner.png",
			LinkUrl:      "https://www.baidu.com",
			MaxShowTimes: 0,
			Platforms:    "all",
			Versions:     "all",
			Sorter:       100,
		},
	}
	return DB.Create(&adverts).Error
}
