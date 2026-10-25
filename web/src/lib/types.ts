export type Me = {
	id: string;
	username: string;
	avatarUrl: string;
};

export type GuildSummary = {
	id: string;
	name: string;
	iconUrl: string;
};

export type Guild = {
	id: string;
	name: string;
	icon: string;
	modLogChannelId: string;
	modRoleId: string;
	adminRoleId: string;
	welcomeChannelId: string;
	welcomeMessage: string;
	autoroleId: string;
	prefix: string;
	reactionChannelId: string;
	reactionMessageId: string;
	updatedAt: string;
};

export type Counts = {
	scheduledAnnouncements: number;
	activeEscalations: number;
	pendingGate: number;
	cases: number;
};

export type Channel = { id: string; name: string; type: number };
export type Role = { id: string; name: string; position: number; managed: boolean };

export type Gate = {
	guildId: string;
	enabled: boolean;
	requiredRoleId: string;
	deadlineSeconds: number;
	warnBeforeSeconds: number;
	kickReason: string;
	dmMessage: string;
	logChannelId: string;
	exemptRoleIds: string[];
	emoji: string;
	channelId: string;
	messageId: string;
	questions: string[];
};

export type Automod = {
	guildId: string;
	enabled: boolean;
	blockInvites: boolean;
	blockSpam: boolean;
	mentionLimit: number;
	capsPercent: number;
	capsMinLength: number;
	badWords: string[];
	action: string;
	timeoutSeconds: number;
	ignoredChannelIds: string[];
	ignoredRoleIds: string[];
};

export type LogConfig = {
	guildId: string;
	channelId: string;
	messageDelete: boolean;
	messageEdit: boolean;
	memberJoin: boolean;
	memberLeave: boolean;
	memberBan: boolean;
};

export type EscalationRole = {
	guildId: string;
	roleId: string;
	tier: string;
	maxDurationSeconds: number;
};

export type Starboard = {
	guildId: string;
	enabled: boolean;
	channelId: string;
	emoji: string;
	threshold: number;
	selfStar: boolean;
	nsfw: boolean;
	ignoredChannelIds: string[];
};

export type GuildEmoji = {
	id: string;
	name: string;
	animated: boolean;
};

export type Bundle = {
	guild: Guild;
	counts: Counts;
	nextKick: string | null;
	channels: Channel[];
	roles: Role[];
	emojis: GuildEmoji[];
	gate: Gate;
	automod: Automod;
	logging: LogConfig;
	escalationRoles: EscalationRole[];
	starboard: Starboard;
};

export type ModCase = {
	id: number;
	caseNumber: number;
	action: string;
	targetId: string;
	targetName: string;
	moderatorName: string;
	reason: string;
	createdAt: string;
};

export type Announcement = {
	id: number;
	channelId: string;
	authorName: string;
	title: string;
	body: string;
	pingRoleId: string;
	scheduledAt?: string;
	sentAt?: string;
	status: string;
	error?: string;
	createdAt: string;
};

export type AnnouncementTemplate = {
	id: number;
	name: string;
	title: string;
	body: string;
	updatedAt: string;
};

export type Escalation = {
	id: number;
	userId: string;
	userName: string;
	roleId: string;
	tier: string;
	grantedByName: string;
	reason: string;
	expiresAt: string;
	status: string;
};

export type GateMember = {
	userId: string;
	userName: string;
	joinedAt: string;
	deadlineAt: string;
	status: string;
};

export type ReactionRole = {
	id: number;
	channelId: string;
	messageId: string;
	emoji: string;
	roleId: string;
};

export type CustomCommand = {
	name: string;
	response: string;
	kind: string;
	roleIds: string[];
};

export type BotProfile = {
	nick: string;
	username: string;
	avatarUrl: string;
	defaultAvatarUrl: string;
	customAvatar: boolean;
};

export type MemberHit = {
	id: string;
	username: string;
	displayName: string;
};
