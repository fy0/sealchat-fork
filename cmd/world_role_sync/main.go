package main

import (
	"log"

	"sealchat/model"
	"sealchat/service"
	"sealchat/utils"
)

func main() {
	config := utils.ReadConfig()
	model.DBInit(config)
	defer func() {
		if db := model.GetDB(); db != nil {
			if sqlDB, err := db.DB(); err == nil {
				_ = sqlDB.Close()
			}
		}
	}()

	if err := service.BackfillWorldRoleAssignments(); err != nil {
		log.Fatalf("回填世界角色关联失败: %v", err)
	}
	log.Println("世界管理员/旁观者通道权限回填完成")
}
