package spec

import "time"

// Define constants for both API and Hosts
const (
	HeartbeatInterval time.Duration = time.Second * 15

	JavaMinecraftDockerImage string = "itzg/minecraft-server"
	JavaMinecraftTCPPort     string = "25565"
)
