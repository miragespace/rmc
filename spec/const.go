package spec

import "time"

// Define constants for both API and Hosts
const (
	HeartbeatInterval time.Duration = time.Second * 15

	JavaMinecraftDockerImage    string = "itzg/minecraft-server"
	JavaMinecraftTCPPort        string = "25565"
	BedrockMinecraftDockerImage string = "itzg/minecraft-bedrock-server"
	BedrockMinecraftUDPPort     string = "19132"
)

type TaskType string

const (
	SubscriptionTask TaskType = "subscription"
)
