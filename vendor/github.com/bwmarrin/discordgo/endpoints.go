// Discordgo - Discord bindings for Go
// Available at https://github.com/bwmarrin/discordgo

// Copyright 2015-2016 Bruce Marriner <bruce@sqls.net>.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file contains variables for all known Discord end points.  All functions
// throughout the Discordgo package use these variables for all connections
// to Discord.  These are all exported and you may modify them if needed.

package discordgo

import "strconv"

// APIVersion is the Discord API version used for the REST and Websocket API.
var APIVersion = "9"

// Known Discord API Endpoints.
var (
	EndpointStatus     = "https://status.discord.com/api/v2/"
	EndpointSm         = EndpointStatus + "scheduled-maintenances/"
	EndpointSmActive   = EndpointSm + "active.json"
	EndpointSmUpcoming = EndpointSm + "upcoming.json"

	EndpointDiscord        = "https://discord.com/"
	EndpointAPI            = EndpointDiscord + "api/v" + APIVersion + "/"
	EndpointGuilds         = EndpointAPI + "guilds/"
	EndpointChannels       = EndpointAPI + "channels/"
	EndpointUsers          = EndpointAPI + "users/"
	EndpointGateway        = EndpointAPI + "gateway"
	EndpointGatewayBot     = EndpointGateway + "/bot"
	EndpointWebhooks       = EndpointAPI + "webhooks/"
	EndpointStickers       = EndpointAPI + "stickers/"
	EndpointStageInstances = EndpointAPI + "stage-instances"
	EndpointSKUs           = EndpointAPI + "skus"

	EndpointCDN             = "https://cdn.discordapp.com/"
	EndpointCDNAttachments  = EndpointCDN + "attachments/"
	EndpointCDNAvatars      = EndpointCDN + "avatars/"
	EndpointCDNIcons        = EndpointCDN + "icons/"
	EndpointCDNSplashes     = EndpointCDN + "splashes/"
	EndpointCDNChannelIcons = EndpointCDN + "channel-icons/"
	EndpointCDNBanners      = EndpointCDN + "banners/"
	EndpointCDNGuilds       = EndpointCDN + "guilds/"
	EndpointCDNRoleIcons    = EndpointCDN + "role-icons/"

	EndpointVoice        = EndpointAPI + "/voice/"
	EndpointVoiceRegions = EndpointVoice + "regions"

	EndpointUser               = func(uID string) string { return EndpointUsers + uID }
	EndpointUserAvatar         = func(uID, aID string) string { return EndpointCDNAvatars + uID + "/" + aID + ".png" }
	EndpointUserAvatarAnimated = func(uID, aID string) string { return EndpointCDNAvatars + uID + "/" + aID + ".gif" }
	EndpointDefaultUserAvatar  = func(idx int) string {
		return EndpointCDN + "embed/avatars/" + strconv.Itoa(idx) + ".png"
	}
	EndpointUserBanner = func(uID, cID string) string {
		return EndpointCDNBanners + uID + "/" + cID + ".png"
	}
	EndpointUserBannerAnimated = func(uID, cID string) string {
		return EndpointCDNBanners + uID + "/" + cID + ".gif"
	}

	EndpointUserGuilds                    = func(uID string) string { return EndpointUsers + uID + "/guilds" }
	EndpointUserGuild                     = func(uID, gID string) string { return EndpointUsers + uID + "/guilds/" + gID }
	EndpointUserGuildMember               = func(uID, gID string) string { return EndpointUserGuild(uID, gID) + "/member" }
	EndpointUserChannels                  = func(uID string) string { return EndpointUsers + uID + "/channels" }
	EndpointUserApplicationRoleConnection = func(aID string) string { return EndpointUsers + "@me/applications/" + aID + "/role-connection" }
	EndpointUserConnections               = func(uID string) string { return EndpointUsers + uID + "/connections" }

	EndpointGuild                    = func(gID string) string { return EndpointGuilds + gID }
	EndpointGuildAutoModeration      = func(gID string) string { return EndpointGuild(gID) + "/auto-moderation" }
	EndpointGuildAutoModerationRules = func(gID string) string { return EndpointGuildAutoModeration(gID) + "/rules" }
	EndpointGuildAutoModerationRule  = func(gID, rID string) string { return EndpointGuildAutoModerationRules(gID) + "/" + rID }
	EndpointGuildThreads             = func(gID string) string { return EndpointGuild(gID) + "/threads" }
	EndpointGuildActiveThreads       = func(gID string) string { return EndpointGuildThreads(gID) + "/active" }
	EndpointGuildPreview             = func(gID string) string { return EndpointGuilds + gID + "/preview" }
	EndpointGuildChannels            = func(gID string) string { return EndpointGuilds + gID + "/channels" }
	EndpointGuildMembers             = func(gID string) string { return EndpointGuilds + gID + "/members" }
	EndpointGuildMembersSearch       = func(gID string) string { return EndpointGuildMembers(gID) + "/search" }
	EndpointGuildMember              = func(gID, uID string) string { return EndpointGuilds + gID + "/members/" + uID }
	EndpointGuildMemberRole          = func(gID, uID, rID string) string { return EndpointGuilds + gID + "/members/" + uID + "/roles/" + rID }
	EndpointGuildBans                = func(gID string) string { return EndpointGuilds + gID + "/bans" }
	EndpointGuildBan                 = func(gID, uID string) string { return EndpointGuilds + gID + "/bans/" + uID }
	EndpointGuildIntegrations        = func(gID string) string { return EndpointGuilds + gID + "/integrations" }
	EndpointGuildIntegration         = func(gID, iID string) string { return EndpointGuilds + gID + "/integrations/" + iID }
	EndpointGuildRoles               = func(gID string) string { return EndpointGuilds + gID + "/roles" }
	EndpointGuildRole                = func(gID, rID string) string { return EndpointGuilds + gID + "/roles/" + rID }
	EndpointGuildInvites             = func(gID string) string { return EndpointGuilds + gID + "/invites" }
	EndpointGuildWidget              = func(gID string) string { return EndpointGuilds + gID + "/widget" }
	EndpointGuildEmbed               = EndpointGuildWidget
	EndpointGuildPrune               = func(gID string) string { return EndpointGuilds + gID + "/prune" }
	EndpointGuildIcon                = func(gID, hash string) string { return EndpointCDNIcons + gID + "/" + hash + ".png" }
	EndpointGuildIconAnimated        = func(gID, hash string) string { return EndpointCDNIcons + gID + "/" + hash + ".gif" }
	EndpointGuildSplash              = func(gID, hash string) string { return EndpointCDNSplashes + gID + "/" + hash + ".png" }
	EndpointGuildWebhooks            = func(gID string) string { return EndpointGuilds + gID + "/webhooks" }
	EndpointGuildAuditLogs           = func(gID string) string { return EndpointGuilds + gID + "/audit-logs" }
	EndpointGuildEmojis              = func(gID string) string { return EndpointGuilds + gID + "/emojis" }
	EndpointGuildEmoji               = func(gID, eID string) string { return EndpointGuilds + gID + "/emojis/" + eID }
	EndpointGuildBanner              = func(gID, hash string) string { return EndpointCDNBanners + gID + "/" + hash + ".png" }
	EndpointGuildBannerAnimated      = func(gID, hash string) string { return EndpointCDNBanners + gID + "/" + hash + ".gif" }
	EndpointGuildStickers            = func(gID string) string { return EndpointGuilds + gID + "/stickers" }
	EndpointGuildSticker             = func(gID, sID string) string { return EndpointGuilds + gID + "/stickers/" + sID }
	EndpointStageInstance            = func(cID string) string { return EndpointStageInstances + "/" + cID }
	EndpointGuildScheduledEvents     = func(gID string) string { return EndpointGuilds + gID + "/scheduled-events" }
	EndpointGuildScheduledEvent      = func(gID, eID string) string { return EndpointGuilds + gID + "/scheduled-events/" + eID }
	EndpointGuildScheduledEventUsers = func(gID, eID string) string { return EndpointGuildScheduledEvent(gID, eID) + "/users" }
	EndpointGuildOnboarding          = func(gID string) string { return EndpointGuilds + gID + "/onboarding" }
	EndpointGuildTemplate            = func(tID string) string { return EndpointGuilds + "templates/" + tID }
	EndpointGuildTemplates           = func(gID string) string { return EndpointGuilds + gID + "/templates" }
	EndpointGuildTemplateSync        = func(gID, tID string) string { return EndpointGuilds + gID + "/templates/" + tID }
	EndpointGuildMemberAvatar        = func(gId, uID, aID string) string {
		return EndpointCDNGuilds + gId + "/users/" + uID + "/avatars/" + aID + ".png"
	}
	EndpointGuildMemberAvatarAnimated = func(gId, uID, aID string) string {
		return EndpointCDNGuilds + gId + "/users/" + uID + "/avatars/" + aID + ".gif"
	}
	EndpointGuildMemberBanner = func(gId, uID, hash string) string {
		return EndpointCDNGuilds + gId + "/users/" + uID + "/banners/" + hash + ".png"
	}
	EndpointGuildMemberBannerAnimated = func(gId, uID, hash string) string {
		return EndpointCDNGuilds + gId + "/users/" + uID + "/banners/" + hash + ".gif"
	}

	EndpointRoleIcon = func(rID, hash string) string {
		return EndpointCDNRoleIcons + rID + "/" + hash + ".png"
	}

	EndpointChannel                             = func(cID string) string { return EndpointChannels + cID }
	EndpointChannelThreads                      = func(cID string) string { return EndpointChannel(cID) + "/threads" }
	EndpointChannelActiveThreads                = func(cID string) string { return EndpointChannelThreads(cID) + "/active" }
	EndpointChannelPublicArchivedThreads        = func(cID string) string { return EndpointChannelThreads(cID) + "/archived/public" }
	EndpointChannelPrivateArchivedThreads       = func(cID string) string { return EndpointChannelThreads(cID) + "/archived/private" }
	EndpointChannelJoinedPrivateArchivedThreads = func(cID string) string { return EndpointChannel(cID) + "/users/@me/threads/archived/private" }
	EndpointChannelPermissions                  = func(cID string) string { return EndpointChannels + cID + "/permissions" }
	EndpointChannelPermission                   = func(cID, tID string) string { return EndpointChannels + cID + "/permissions/" + tID }
	EndpointChannelInvites                      = func(cID string) string { return EndpointChannels + cID + "/invites" }
	EndpointChannelTyping                       = func(cID string) string { return EndpointChannels + cID + "/typing" }
	EndpointChannelMessages                     = func(cID string) string { return EndpointChannels + cID + "/messages" }
	EndpointChannelMessage                      = func(cID, mID string) string { return EndpointChannels + cID + "/messages/" + mID }
	EndpointChannelMessageThread                = func(cID, mID string) string { return EndpointChannelMessage(cID, mID) + "/threads" }
	EndpointChannelMessagesBulkDelete           = func(cID string) string { return EndpointChannel(cID) + "/messages/bulk-delete" }
	EndpointChannelMessagesPins                 = func(cID string) string { return EndpointChannel(cID) + "/pins" }
	EndpointChannelMessagePin                   = func(cID, mID string) string { return EndpointChannel(cID) + "/pins/" + mID }
	EndpointChannelMessageCrosspost             = func(cID, mID string) string { return EndpointChannel(cID) + "/messages/" + mID + "/crosspost" }
	EndpointChannelFollow                       = func(cID string) string { return EndpointChannel(cID) + "/followers" }
	EndpointThreadMembers                       = func(tID string) string { return EndpointChannel(tID) + "/thread-members" }
	EndpointThreadMember                        = func(tID, mID string) string { return EndpointThreadMembers(tID) + "/" + mID }

	EndpointGroupIcon = func(cID, hash string) string { return EndpointCDNChannelIcons + cID + "/" + hash + ".png" }

	EndpointSticker            = func(sID string) string { return EndpointStickers + sID }
	EndpointNitroStickersPacks = EndpointAPI + "/sticker-packs"

	EndpointChannelWebhooks = func(cID string) string { return EndpointChannel(cID) + "/webhooks" }
	EndpointWebhook         = func(wID string) string { return EndpointWebhooks + wID }
	EndpointWebhookToken    = func(wID, token string) string { return EndpointWebhooks + wID + "/" + token }
	EndpointWebhookMessage  = func(wID, token, messageID string) string {
		return EndpointWebhookToken(wID, token) + "/messages/" + messageID
	}

	EndpointMessageReactionsAll = func(cID, mID string) string {
		return EndpointChannelMessage(cID, mID) + "/reactions"
	}
	EndpointMessageReactions = func(cID, mID, eID string) string {
		return EndpointChannelMessage(cID, mID) + "/reactions/" + eID
	}
	EndpointMessageReaction = func(cID, mID, eID, uID string) string {
		return EndpointMessageReactions(cID, mID, eID) + "/" + uID
	}

	EndpointPoll = func(cID, mID string) string {
		return EndpointChannel(cID) + "/polls/" + mID
	}
	EndpointPollAnswerVoters = func(cID, mID string, aID int) string {
		return EndpointPoll(cID, mID) + "/answers/" + strconv.Itoa(aID)
	}
	EndpointPollExpire = func(cID, mID string) string {
		return EndpointPoll(cID, mID) + "/expire"
	}

	EndpointApplicationSKUs = func(aID string) string {
		return EndpointApplication(aID) + "/skus"
	}

	EndpointEntitlements = func(aID string) string {
		return EndpointApplication(aID) + "/entitlements"
	}
	EndpointEntitlement = func(aID, eID string) string {
		return EndpointEntitlements(aID) + "/" + eID
	}
	EndpointEntitlementConsume = func(aID, eID string) string {
		return EndpointEntitlement(aID, eID) + "/consume"
	}

	EndpointSubscriptions = func(skuID string) string {
		return EndpointSKUs + "/" + skuID + "/subscriptions"
	}
	EndpointSubscription = func(skuID, subID string) string {
		return EndpointSubscriptions(skuID) + "/" + subID
	}

	EndpointApplicationGlobalCommands = func(aID string) string {
		return EndpointApplication(aID) + "/commands"
	}
	EndpointApplicationGlobalCommand = func(aID, cID string) string {
		return EndpointApplicationGlobalCommands(aID) + "/" + cID
	}

	EndpointApplicationGuildCommands = func(aID, gID string) string {
		return EndpointApplication(aID) + "/guilds/" + gID + "/commands"
	}
	EndpointApplicationGuildCommand = func(aID, gID, cID string) string {
		return EndpointApplicationGuildCommands(aID, gID) + "/" + cID
	}
	EndpointApplicationCommandPermissions = func(aID, gID, cID string) string {
		return EndpointApplicationGuildCommand(aID, gID, cID) + "/permissions"
	}
	EndpointApplicationCommandsGuildPermissions = func(aID, gID string) string {
		return EndpointApplicationGuildCommands(aID, gID) + "/permissions"
	}
	EndpointInteraction = func(aID, iToken string) string {
		return EndpointAPI + "interactions/" + aID + "/" + iToken
	}
	EndpointInteractionResponse = func(iID, iToken string) string {
		return EndpointInteraction(iID, iToken) + "/callback"
	}
	EndpointInteractionResponseActions = func(aID, iToken string) string {
		return EndpointWebhookMessage(aID, iToken, "@original")
	}
	EndpointFollowupMessage = func(aID, iToken string) string {
		return EndpointWebhookToken(aID, iToken)
	}
	EndpointFollowupMessageActions = func(aID, iToken, mID string) string {
		return EndpointWebhookMessage(aID, iToken, mID)
	}

	EndpointGuildCreate = EndpointAPI + "guilds"

	EndpointInvite = func(iID string) string { return EndpointAPI + "invites/" + iID }

	EndpointEmoji         = func(eID string) string { return EndpointCDN + "emojis/" + eID + ".png" }
	EndpointEmojiAnimated = func(eID string) string { return EndpointCDN + "emojis/" + eID + ".gif" }

	EndpointApplications                      = EndpointAPI + "applications"
	EndpointApplication                       = func(aID string) string { return EndpointApplications + "/" + aID }
	EndpointApplicationRoleConnectionMetadata = func(aID string) string { return EndpointApplication(aID) + "/role-connections/metadata" }

	EndpointApplicationEmojis = func(aID string) string { return EndpointApplication(aID) + "/emojis" }
	EndpointApplicationEmoji  = func(aID, eID string) string { return EndpointApplication(aID) + "/emojis/" + eID }

	EndpointOAuth2                  = EndpointAPI + "oauth2/"
	EndpointOAuth2Applications      = EndpointOAuth2 + "applications"
	EndpointOAuth2Application       = func(aID string) string { return EndpointOAuth2Applications + "/" + aID }
	EndpointOAuth2ApplicationsBot   = func(aID string) string { return EndpointOAuth2Applications + "/" + aID + "/bot" }
	EndpointOAuth2ApplicationAssets = func(aID string) string { return EndpointOAuth2Applications + "/" + aID + "/assets" }

	// TODO: Deprecated, remove in the next release
	EndpointOauth2                  = EndpointOAuth2
	EndpointOauth2Applications      = EndpointOAuth2Applications
	EndpointOauth2Application       = EndpointOAuth2Application
	EndpointOauth2ApplicationsBot   = EndpointOAuth2ApplicationsBot
	EndpointOauth2ApplicationAssets = EndpointOAuth2ApplicationAssets
)
