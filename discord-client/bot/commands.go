package bot

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// registerCommands registers all slash commands with Discord
func (b *Bot) registerCommands() error {
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "balance",
			Description: "Check your current balance",
		},
		{
			Name:        "gamble",
			Description: "Open the interactive betting interface",
		},
		{
			Name:        "donate",
			Description: "Transfer bits to another player",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "amount",
					Description: "Amount to donate in bits",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "User to donate to",
					Required:    true,
				},
			},
		},
		{
			Name:        "wager",
			Description: "Create and manage wagers",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "propose",
					Description: "Propose a wager against another player",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user",
							Description: "User to wager against",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "amount",
							Description: "Amount to wager in bits",
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "list",
					Description: "List your active wagers",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "cancel",
					Description: "Cancel a proposed wager",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "id",
							Description: "Wager ID to cancel",
							Required:    true,
						},
					},
				},
			},
		},
		{
			Name:        "groupwager",
			Description: "Create and manage group wagers",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "create",
					Description: "Create a new group wager (opens modal for details)",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "resolve",
					Description: "Resolve a group wager (resolvers only)",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "id",
							Description: "Group wager ID to resolve",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "winning_option",
							Description: "Exact text of the winning option",
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "cancel",
					Description: "Cancel an active group wager",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "id",
							Description: "Group wager ID to cancel",
							Required:    true,
						},
					},
				},
			},
		},
		{
			Name:        "stats",
			Description: "View player statistics",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "scoreboard",
					Description: "Display the top players scoreboard",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "balance",
					Description: "Display detailed statistics for a player",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user",
							Description: "User to check stats for (defaults to you)",
							Required:    false,
						},
					},
				},
			},
		},
		{
			Name:        "settings",
			Description: "Configure guild settings (admin only)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "high-roller-role",
					Description: "Set the role assigned to the player with the most bits",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionRole,
							Name:        "role",
							Description: "The role to assign to the high roller (leave empty to disable)",
							Required:    false,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "primary-channel",
					Description: "Set the primary channel for bot activities",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "channel",
							Description: "The channel to set as primary (leave empty to disable)",
							Required:    false,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "lol-channel",
					Description: "Set the channel for League of Legends activities",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "channel",
							Description: "The channel to set for LOL activities (leave empty to disable)",
							Required:    false,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "tft-channel",
					Description: "Set the channel for Teamfight Tactics activities",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "channel",
							Description: "The channel to set for TFT activities (leave empty to disable)",
							Required:    false,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "wordle-channel",
					Description: "Set the channel for Wordle results and rewards",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "channel",
							Description: "The channel for Wordle activities (leave empty to disable)",
							Required:    false,
						},
					},
				},
			},
		},
		{
			Name:        "summoner",
			Description: "League of Legends summoner tracking",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "watch",
					Description: "Start tracking a summoner",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "game_name",
							Description: "League of Legends game name (e.g., Faker)",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "tag",
							Description: "Riot ID tag line (e.g., KR1)",
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "unwatch",
					Description: "Stop tracking a summoner",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "game_name",
							Description: "League of Legends game name (e.g., Faker)",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "tag",
							Description: "Riot ID tag line (e.g., KR1)",
							Required:    true,
						},
					},
				},
			},
		},
	}

	for _, cmd := range commands {
		_, err := b.session.ApplicationCommandCreate(b.session.State.User.ID, "", cmd)
		if err != nil {
			return fmt.Errorf("cannot create '%s' command: %w", cmd.Name, err)
		}
	}

	return nil
}
