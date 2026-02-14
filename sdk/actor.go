package sdk

import "github.com/ZaiSpace/nexo_im/common"

func GetUserActorUserId(userId int64) string {
	ac := common.Actor{Id: userId, Role: common.RoleUser}
	ret, _ := ac.ToIMUserId()
	return ret
}

func GetAgentActorUserId(agentId int64) string {
	ac := common.Actor{Id: agentId, Role: common.RoleAgent}
	ret, _ := ac.ToIMUserId()
	return ret
}

func MGetUserActorUserIds(userIds []int64) []string {
	var ret []string
	for _, userId := range userIds {
		ret = append(ret, GetUserActorUserId(userId))
	}
	return ret
}

func MGetAgentActorUserIds(agentIds []int64) []string {
	var ret []string
	for _, agentId := range agentIds {
		ret = append(ret, GetAgentActorUserId(agentId))
	}
	return ret
}

func GetActorFromUserId(userId string) (*common.Actor, error) {
	a := new(common.Actor)
	err := a.FromIMUserId(userId)
	return a, err
}

func MGetActorFromUserIds(userIds []string) ([]*common.Actor, error) {
	var ret []*common.Actor
	for _, userId := range userIds {
		a := new(common.Actor)
		err := a.FromIMUserId(userId)
		if err != nil {
			return nil, err
		}
	}
	return ret, nil
}
