package logic

import (
	"context"

	"userop_srv/global"
	"userop_srv/model"
	"userop_srv/proto"
)

// MessageList 留言板列表
func (*UserOpServer) MessageList(ctx context.Context, req *proto.MessageRequest) (*proto.MessageListResponse, error) {
	var rsp proto.MessageListResponse
	var messages []model.LeavingMessages
	var messageList []*proto.MessageResponse

	result := global.MysqlConf.DB.Where(&model.LeavingMessages{User: req.UserId}).Find(&messages)
	rsp.Total = int32(result.RowsAffected)

	for _, message := range messages {
		messageList = append(messageList, &proto.MessageResponse{
			Id:          message.ID,
			UserId:      message.User,
			MessageType: message.MessageType,
			Subject:     message.Subject,
			Message:     message.Message,
			File:        message.File,
		})
	}

	rsp.Data = messageList
	return &rsp, nil
}

// CreateMessage 创建留言板内容
func (*UserOpServer) CreateMessage(ctx context.Context, req *proto.MessageRequest) (*proto.MessageResponse, error) {
	var message model.LeavingMessages

	message.User = req.UserId
	message.MessageType = req.MessageType
	message.Subject = req.Subject
	message.Message = req.Message
	message.File = req.File

	global.MysqlConf.DB.Save(&message)

	return &proto.MessageResponse{Id: message.ID}, nil
}
