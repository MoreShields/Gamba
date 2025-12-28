#!/usr/bin/env python3
"""
Year in Review Graphic Generator

Usage: ./generate-graphic.py stats.json output.png [--scale N]
  --scale: Resolution multiplier (1-4, default 2 for 2x/retina)

Requires: Pillow, httpx
Env:
  DISCORD_TOKEN - for username resolution (auto-loads from repo root .env)
  GUILD_ID - for role color resolution (optional, requires DISCORD_TOKEN)
"""

import json
import os
import sys
from typing import Optional

try:
    from PIL import Image, ImageDraw, ImageFont
except ImportError:
    print("Error: Pillow is required. Install with: pip install Pillow", file=sys.stderr)
    sys.exit(1)

try:
    import httpx
except ImportError:
    print("Error: httpx is required. Install with: pip install httpx", file=sys.stderr)
    sys.exit(1)


# Graphic configuration (base values, scaled at runtime)
BASE_WIDTH = 600
BASE_PADDING = 40
BASE_SECTION_SPACING = 30
BASE_STAT_SPACING = 20
BASE_HEADER_HEIGHT = 120

# Colors
BG_COLOR = (24, 24, 32)  # Dark background
HEADER_BG = (40, 40, 60)
TEXT_COLOR = (255, 255, 255)
ACCENT_COLOR = (100, 200, 255)
SECONDARY_TEXT = (200, 200, 210)
SECTION_BG = (32, 32, 45)
HIGHLIGHT_COLOR = (255, 215, 0)  # Gold
BASE_INDENT = 20


def int_to_rgb(color_int: int) -> tuple[int, int, int]:
    """Convert Discord color int to RGB tuple."""
    return ((color_int >> 16) & 255, (color_int >> 8) & 255, color_int & 255)


def ensure_visible(color: tuple[int, int, int], min_brightness: int = 90) -> tuple[int, int, int]:
    """Lighten colors that are too dark to be visible on dark backgrounds."""
    r, g, b = color
    # Calculate perceived brightness (weighted for human perception)
    brightness = (r * 299 + g * 587 + b * 114) // 1000

    if brightness < min_brightness:
        # Calculate how much to boost
        if brightness == 0:
            # Pure black - return a neutral gray
            return (min_brightness, min_brightness, min_brightness)

        factor = min_brightness / brightness
        # Boost while preserving color ratios, cap at 255
        r = min(255, int(r * factor))
        g = min(255, int(g * factor))
        b = min(255, int(b * factor))

    return (r, g, b)


class DiscordUserResolver:
    """Resolves Discord user IDs to usernames and role colors via API."""

    def __init__(self, token: str, guild_id: Optional[str] = None):
        self.token = token
        self.guild_id = guild_id
        # Cache stores (username, color) tuples
        self.cache: dict[str, tuple[str, tuple[int, int, int]]] = {}
        self.roles_cache: Optional[list[dict]] = None

    def _fetch_roles(self) -> list[dict]:
        """Fetch and cache guild roles, sorted by position (highest first)."""
        if self.roles_cache is not None:
            return self.roles_cache

        if not self.guild_id:
            self.roles_cache = []
            return self.roles_cache

        try:
            with httpx.Client() as client:
                response = client.get(
                    f"https://discord.com/api/v10/guilds/{self.guild_id}/roles",
                    headers={"Authorization": f"Bot {self.token}"},
                    timeout=10.0,
                )
                if response.status_code == 200:
                    roles = response.json()
                    # Sort by position descending (highest first)
                    self.roles_cache = sorted(roles, key=lambda r: r.get("position", 0), reverse=True)
                    return self.roles_cache
        except Exception as e:
            print(f"Warning: Failed to fetch guild roles: {e}", file=sys.stderr)

        self.roles_cache = []
        return self.roles_cache

    def _get_member_color(self, discord_id: str) -> Optional[tuple[int, int, int]]:
        """Get user's highest colored role as RGB tuple."""
        if not self.guild_id:
            return None

        try:
            with httpx.Client() as client:
                response = client.get(
                    f"https://discord.com/api/v10/guilds/{self.guild_id}/members/{discord_id}",
                    headers={"Authorization": f"Bot {self.token}"},
                    timeout=10.0,
                )
                if response.status_code == 200:
                    member = response.json()
                    member_role_ids = set(member.get("roles", []))

                    # Get roles sorted by position
                    roles = self._fetch_roles()

                    # Find highest-positioned role with a color
                    for role in roles:
                        if role.get("id") in member_role_ids:
                            color = role.get("color", 0)
                            if color != 0:
                                return int_to_rgb(color)
        except Exception as e:
            print(f"Warning: Failed to fetch member {discord_id}: {e}", file=sys.stderr)

        return None

    def _fetch_username(self, discord_id: str) -> str:
        """Fetch username from Discord API."""
        try:
            with httpx.Client() as client:
                response = client.get(
                    f"https://discord.com/api/v10/users/{discord_id}",
                    headers={"Authorization": f"Bot {self.token}"},
                    timeout=10.0,
                )
                if response.status_code == 200:
                    data = response.json()
                    return data.get("global_name") or data.get("username", f"User {discord_id}")
        except Exception as e:
            print(f"Warning: Failed to resolve user {discord_id}: {e}", file=sys.stderr)

        return f"User {discord_id[-4:]}"

    def get_user_info(self, discord_id: str) -> tuple[str, tuple[int, int, int]]:
        """Get username and color for a Discord ID, with caching."""
        if discord_id in self.cache:
            return self.cache[discord_id]

        username = self._fetch_username(discord_id)
        color = self._get_member_color(discord_id) or TEXT_COLOR
        # Ensure dark colors are lightened for visibility
        color = ensure_visible(color)

        self.cache[discord_id] = (username, color)
        return username, color


def load_font(size: int, bold: bool = False) -> ImageFont.FreeTypeFont | ImageFont.ImageFont:
    """Load a font, falling back to default if needed."""
    font_paths = [
        "/System/Library/Fonts/Helvetica.ttc",
        "/System/Library/Fonts/SFNSText.ttf",
        "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
        "/usr/share/fonts/TTF/DejaVuSans.ttf",
    ]

    if bold:
        font_paths = [
            "/System/Library/Fonts/Helvetica.ttc",
            "/System/Library/Fonts/SFNSText-Bold.ttf",
            "/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf",
        ] + font_paths

    for path in font_paths:
        if os.path.exists(path):
            try:
                return ImageFont.truetype(path, size)
            except Exception:
                continue

    return ImageFont.load_default()


class InfographicRenderer:
    """Renders the year-in-review infographic."""

    def __init__(self, stats: dict, resolver: Optional[DiscordUserResolver], scale: int = 1):
        self.stats = stats
        self.resolver = resolver
        self.scale = scale
        # Scaled dimensions
        self.width = BASE_WIDTH * scale
        self.padding = BASE_PADDING * scale
        self.section_spacing = BASE_SECTION_SPACING * scale
        self.header_height = BASE_HEADER_HEIGHT * scale
        # Scaled fonts
        self.font_title = load_font(48 * scale, bold=True)
        self.font_section = load_font(28 * scale, bold=True)
        self.font_subheading = load_font(22 * scale, bold=True)
        self.font_value = load_font(18 * scale, bold=True)
        self.font_detail = load_font(14 * scale)

    def resolve_user(self, discord_id: Optional[str]) -> tuple[str, tuple[int, int, int]]:
        """Resolve a Discord ID to (username, color) tuple."""
        if not discord_id:
            return ("Unknown", TEXT_COLOR)
        if self.resolver:
            return self.resolver.get_user_info(str(discord_id))
        return (f"User {str(discord_id)[-4:]}", TEXT_COLOR)

    def calculate_height(self) -> int:
        """Calculate total image height needed.

        Entry spacing breakdown (base values, scaled by self.scale):
        - With subheading + detail: 24 + 20 + 32 = 76px
        - With subheading, no detail: 24 + 20 + 16 = 60px
        - No subheading + detail: 20 + 32 = 52px
        """
        height = self.header_height + self.padding * 2
        s = self.scale

        # Server activity section (28 + 40 + section_spacing)
        if self.stats.get("total_server_activity"):
            height += (68 * s) + self.section_spacing

        # (key, entries_with_subheading_and_detail, entries_with_subheading_only, entries_no_subheading_with_detail)
        sections = [
            ("gambling", 4, 2, 8),
            ("group_wagers", 1, 3, 0),
            ("wordle_stats", 3, 1, 0),
            ("high_roller", 1, 0, 2),
            ("lol", 3, 1, 4),
            ("tft", 3, 1, 4),
        ]

        for key, full_entries, subheading_only, no_subheading in sections:
            if key in self.stats and self.stats[key]:
                entry_height = (full_entries * 76 * s) + (subheading_only * 60 * s) + (no_subheading * 52 * s)
                height += (50 * s) + entry_height + self.section_spacing

        return height + self.padding

    def draw_header(self, draw: ImageDraw.ImageDraw, y: int) -> int:
        """Draw the header section."""
        title = "Gamba Season 1 Review"
        bbox = draw.textbbox((0, 0), title, font=self.font_title)
        title_width = bbox[2] - bbox[0]
        draw.text(
            ((self.width - title_width) // 2, y),
            title,
            font=self.font_title,
            fill=TEXT_COLOR,
        )
        y += 60 * self.scale
        return y + 40 * self.scale

    def draw_server_activity(self, draw: ImageDraw.ImageDraw, y: int) -> int:
        """Draw the total server activity stat at the top."""
        activity = self.stats.get("total_server_activity", {})
        if not activity:
            return y

        s = self.scale
        total_volume = activity.get("total_volume", 0)

        # Draw label
        label = "Total Server Activity"
        bbox = draw.textbbox((0, 0), label, font=self.font_subheading)
        label_width = bbox[2] - bbox[0]
        draw.text(((self.width - label_width) // 2, y), label, font=self.font_subheading, fill=ACCENT_COLOR)
        y += 28 * s

        # Draw value
        value = f"{self.format_number(total_volume)} bits"
        bbox = draw.textbbox((0, 0), value, font=self.font_section)
        value_width = bbox[2] - bbox[0]
        draw.text(((self.width - value_width) // 2, y), value, font=self.font_section, fill=HIGHLIGHT_COLOR)
        y += 40 * s

        return y + self.section_spacing

    def _calculate_section_height(self, stats: list[tuple]) -> int:
        """Calculate the height needed for a section's content."""
        s = self.scale
        height = 50 * s  # Section title
        for stat in stats:
            stat_name = stat[0]
            stat_value = stat[1]
            stat_detail = stat[2] if len(stat) > 2 else None
            if stat_value:
                if stat_name:
                    height += 24 * s  # Subheading
                height += 20 * s  # Value
                if stat_detail:
                    height += 32 * s  # Detail + spacing
                else:
                    height += 16 * s  # No detail spacing
            else:
                height += 24 * s
        return height

    def draw_section(
        self, draw: ImageDraw.ImageDraw, y: int, title: str, stats: list[tuple],
        header_color: tuple[int, int, int] = TEXT_COLOR
    ) -> int:
        """Draw a section with stats."""
        s = self.scale
        # Section background - calculate actual height needed
        section_height = self._calculate_section_height(stats)
        draw.rounded_rectangle(
            [self.padding - 10 * s, y, self.width - self.padding + 10 * s, y + section_height],
            radius=10 * s,
            fill=SECTION_BG,
        )

        # Section title (centered)
        bbox = draw.textbbox((0, 0), title, font=self.font_section)
        title_width = bbox[2] - bbox[0]
        draw.text(((self.width - title_width) // 2, y + 10 * s), title, font=self.font_section, fill=header_color)
        y += 50 * s

        # Stats - format: (subheading, value, detail, value_color)
        for stat in stats:
            stat_name = stat[0]
            stat_value = stat[1]
            stat_detail = stat[2] if len(stat) > 2 else None
            value_color = stat[3] if len(stat) > 3 else TEXT_COLOR

            if stat_value:
                # Render subheading if present (centered)
                if stat_name:
                    bbox = draw.textbbox((0, 0), stat_name, font=self.font_subheading)
                    text_width = bbox[2] - bbox[0]
                    draw.text(((self.width - text_width) // 2, y), stat_name, font=self.font_subheading, fill=header_color)
                    y += 24 * s
                # Render value (centered)
                value_text = str(stat_value)
                bbox = draw.textbbox((0, 0), value_text, font=self.font_value)
                text_width = bbox[2] - bbox[0]
                draw.text(((self.width - text_width) // 2, y), value_text, font=self.font_value, fill=value_color)
                y += 20 * s
                # Render detail if present (centered)
                if stat_detail:
                    bbox = draw.textbbox((0, 0), stat_detail, font=self.font_detail)
                    text_width = bbox[2] - bbox[0]
                    draw.text(((self.width - text_width) // 2, y), stat_detail, font=self.font_detail, fill=SECONDARY_TEXT)
                    y += 32 * s
                else:
                    y += 16 * s
            else:
                y += 24 * s

        return y + self.section_spacing

    def format_number(self, num: Optional[int | float]) -> str:
        """Format a number for display."""
        if num is None:
            return "N/A"
        if isinstance(num, float):
            return f"{num:,.2f}"
        return f"{num:,}"

    def render(self) -> Image.Image:
        """Render the full infographic."""
        height = self.calculate_height()
        img = Image.new("RGB", (self.width, height), BG_COLOR)
        draw = ImageDraw.Draw(img)

        y = self.padding
        y = self.draw_header(draw, y)
        y = self.draw_server_activity(draw, y)

        # Gambling Stats
        gambling = self.stats.get("gambling", {})
        if gambling:
            stats = []

            top_wagered = gambling.get("top_bits_wagered", [])
            if top_wagered:
                for i, s in enumerate(top_wagered[:3], 1):
                    user, user_color = self.resolve_user(s.get("discord_id"))
                    label = "Most Bits Wagered" if i == 1 else ""
                    stats.append((
                        label,
                        f"#{i} {user}",
                        f"{self.format_number(s.get('total_wagered'))} bits | {self.format_number(s.get('bet_count'))} bets",
                        user_color
                    ))


            top_winners = gambling.get("top_winners", [])
            if top_winners:
                for i, s in enumerate(top_winners[:3], 1):
                    user, user_color = self.resolve_user(s.get("discord_id"))
                    label = "Biggest Winners" if i == 1 else ""
                    net_profit = s.get('net_profit', 0)
                    stats.append((
                        label,
                        f"#{i} {user}",
                        f"+{self.format_number(net_profit)} bits profit | {self.format_number(s.get('bet_count'))} bets",
                        user_color
                    ))

            top_losers = gambling.get("top_losers", [])
            if top_losers:
                for i, s in enumerate(top_losers[:3], 1):
                    user, user_color = self.resolve_user(s.get("discord_id"))
                    label = "Biggest Losers" if i == 1 else ""
                    net_profit = s.get('net_profit', 0)
                    stats.append((
                        label,
                        f"#{i} {user}",
                        f"{self.format_number(net_profit)} bits | {self.format_number(s.get('bet_count'))} bets",
                        user_color
                    ))

            top_10_hits = gambling.get("top_10_percent_hits", [])
            if top_10_hits:
                for i, s in enumerate(top_10_hits[:3], 1):
                    user, user_color = self.resolve_user(s.get("discord_id"))
                    label = "Most 10% Odds Hits" if i == 1 else ""
                    stats.append((
                        label,
                        f"#{i} {user}",
                        f"{self.format_number(s.get('hit_count'))} wins",
                        user_color
                    ))
            if gambling.get("total_bets_placed") and gambling.get("total_amount_wagered"):
                count = self.format_number(gambling["total_bets_placed"].get("count"))
                amount = self.format_number(gambling['total_amount_wagered'].get('amount'))
                stats.append((
                    "Gambling Totals",
                    f"{count} gambles | {amount} bits",
                    None
                ))

            if gambling.get("most_h2h_wager_wins"):
                s = gambling["most_h2h_wager_wins"]
                user, user_color = self.resolve_user(s.get("discord_id"))
                stats.append((
                    "Most 1v1 Wager Wins",
                    f"{user} - {self.format_number(s.get('win_count'))} wins",
                    None,
                    user_color
                ))



            if stats:
                y = self.draw_section(draw, y, "Gambling Stats", stats)

        # Group Wagers
        gw = self.stats.get("group_wagers", {})
        if gw:
            stats = []

            if gw.get("most_created"):
                s = gw["most_created"]
                user, user_color = self.resolve_user(s.get("discord_id"))
                stats.append((
                    "Most Wagers Created",
                    f"{user} - {self.format_number(s.get('created_count'))} wagers",
                    None,
                    user_color
                ))

            if gw.get("most_wins"):
                s = gw["most_wins"]
                user, user_color = self.resolve_user(s.get("discord_id"))
                stats.append((
                    "Most Group Wager Wins",
                    f"{user} - {self.format_number(s.get('win_count'))} wins",
                    f"+{self.format_number(s.get('total_profit'))} bits profit",
                    user_color
                ))

            if gw.get("most_participation"):
                s = gw["most_participation"]
                user, user_color = self.resolve_user(s.get("discord_id"))
                stats.append((
                    "Most Participation",
                    f"{user} - {self.format_number(s.get('participation_count'))} wagers joined",
                    None,
                    user_color
                ))

            if gw.get("total_group_wagers"):
                count = self.format_number(gw["total_group_wagers"].get("count"))
                amount = self.format_number(gw.get('total_amount_wagered', {}).get('amount'))
                stats.append((
                    "Total Group Wagers",
                    f"{count} wagers | {amount} bits",
                    None
                ))

            if stats:
                y = self.draw_section(draw, y, "Group Wagers", stats)

        # Wordle Stats
        ws = self.stats.get("wordle_stats", {})
        if ws:
            stats = []

            if ws.get("longest_streak"):
                s = ws["longest_streak"]
                user, user_color = self.resolve_user(s.get("discord_id"))
                stats.append((
                    "Longest Streak",
                    f"{user} - {s.get('streak_length', 0)} days",
                    f"{s.get('streak_start')} to {s.get('streak_end')}",
                    user_color
                ))

            if ws.get("most_completions"):
                s = ws["most_completions"]
                user, user_color = self.resolve_user(s.get("discord_id"))
                stats.append((
                    "Most Completions",
                    f"{user} - {self.format_number(s.get('completion_count'))} wordles",
                    None,
                    user_color
                ))

            if ws.get("most_rewards"):
                s = ws["most_rewards"]
                user, user_color = self.resolve_user(s.get("discord_id"))
                stats.append((
                    "Most Rewards Earned",
                    f"{user} - {self.format_number(s.get('total_rewards'))} bits",
                    f"From {s.get('reward_count', 0)} rewards",
                    user_color
                ))
            if ws.get("best_avg_guesses"):
                s = ws["best_avg_guesses"]
                user, user_color = self.resolve_user(s.get("discord_id"))
                stats.append((
                    "Best Average Guesses",
                    f"{user} - {s.get('avg_guesses', 0)} avg",
                    f"Over {s.get('completion_count', 0)} completions",
                    user_color
                ))

            if stats:
                y = self.draw_section(draw, y, "Wordle Stats", stats)

        # High Roller Stats
        hr = self.stats.get("high_roller", {})
        if hr:
            stats = []

            top_spenders = hr.get("top_spenders", [])
            if top_spenders:
                for i, s in enumerate(top_spenders[:3], 1):
                    user, user_color = self.resolve_user(s.get("discord_id"))
                    stats.append((
                        "",
                        f"#{i} {user}",
                        f"{self.format_number(s.get('total_spent'))} bits | {s.get('purchase_count', 0)} purchases",
                        user_color
                    ))

            if stats:
                y = self.draw_section(draw, y, "King Coin", stats)

        # LoL Stats
        lol = self.stats.get("lol", {})
        if lol:
            stats = []

            # Top 3 Most Profitable Summoners
            most_profitable = lol.get("most_profitable_summoners", [])
            if most_profitable:
                for i, s in enumerate(most_profitable[:3], 1):
                    summoner = s.get('summoner', 'Unknown')
                    wager_count = s.get('wager_count', 0)
                    total_wagered = self.format_number(s.get('total_wagered'))
                    net_profit = self.format_number(s.get('net_profit'))
                    # First entry gets the section label
                    label = "Most Profitable Summoners" if i == 1 else ""
                    stats.append((
                        label,
                        f"#{i} {summoner}",
                        f"{wager_count} games | {total_wagered} wagered | +{net_profit} profit"
                    ))

            # Top 3 Least Profitable Summoners
            least_profitable = lol.get("least_profitable_summoners", [])
            if least_profitable:
                for i, s in enumerate(least_profitable[:3], 1):
                    summoner = s.get('summoner', 'Unknown')
                    wager_count = s.get('wager_count', 0)
                    total_wagered = self.format_number(s.get('total_wagered'))
                    net_profit = self.format_number(s.get('net_profit'))
                    # First entry gets the section label
                    label = "Least Profitable Summoners" if i == 1 else ""
                    stats.append((
                        label,
                        f"#{i} {summoner}",
                        f"{wager_count} games | {total_wagered} wagered | {net_profit} loss"
                    ))

            if lol.get("most_bet_on_summoner"):
                s = lol["most_bet_on_summoner"]
                stats.append((
                    "Most Bet On",
                    f"{s.get('summoner', 'Unknown')}",
                    f"{self.format_number(s.get('total_wagered'))} bits wagered over {s.get('wager_count', 0)} games"
                ))

            if lol.get("total_amount_wagered"):
                stats.append((
                    "Total LoL Wagered",
                    f"{self.format_number(lol['total_amount_wagered'].get('amount'))} bits",
                    None
                ))

            if stats:
                y = self.draw_section(draw, y, "League of Legends", stats, ACCENT_COLOR)

        # TFT Stats
        tft = self.stats.get("tft", {})
        if tft:
            stats = []

            # Top 3 Most Profitable Summoners
            most_profitable = tft.get("most_profitable_summoners", [])
            if most_profitable:
                for i, s in enumerate(most_profitable[:3], 1):
                    summoner = s.get('summoner', 'Unknown')
                    wager_count = s.get('wager_count', 0)
                    total_wagered = self.format_number(s.get('total_wagered'))
                    net_profit = self.format_number(s.get('net_profit'))
                    # First entry gets the section label
                    label = "Most Profitable Summoners" if i == 1 else ""
                    stats.append((
                        label,
                        f"#{i} {summoner}",
                        f"{wager_count} games | {total_wagered} wagered | +{net_profit} profit"
                    ))

            # Top 3 Least Profitable Summoners
            least_profitable = tft.get("least_profitable_summoners", [])
            if least_profitable:
                for i, s in enumerate(least_profitable[:3], 1):
                    summoner = s.get('summoner', 'Unknown')
                    wager_count = s.get('wager_count', 0)
                    total_wagered = self.format_number(s.get('total_wagered'))
                    net_profit = self.format_number(s.get('net_profit'))
                    # First entry gets the section label
                    label = "Least Profitable Summoners" if i == 1 else ""
                    stats.append((
                        label,
                        f"#{i} {summoner}",
                        f"{wager_count} games | {total_wagered} wagered | {net_profit} loss"
                    ))

            if tft.get("most_bet_on_summoner"):
                s = tft["most_bet_on_summoner"]
                stats.append((
                    "Most Bet On",
                    f"{s.get('summoner', 'Unknown')}",
                    f"{self.format_number(s.get('total_wagered'))} bits wagered over {s.get('wager_count', 0)} games"
                ))

            if tft.get("total_amount_wagered"):
                stats.append((
                    "Total TFT Wagered",
                    f"{self.format_number(tft['total_amount_wagered'].get('amount'))} bits",
                    None
                ))

            if stats:
                y = self.draw_section(draw, y, "TFT", stats, ACCENT_COLOR)

        return img


def main():
    import argparse

    parser = argparse.ArgumentParser(description="Generate year-in-review infographic")
    parser.add_argument("stats_file", help="Path to stats.json")
    parser.add_argument("output_file", help="Path for output PNG")
    parser.add_argument("--scale", type=int, default=3, choices=[1, 2, 3, 4],
                        help="Scale factor for higher resolution (default: 2)")
    args = parser.parse_args()

    stats_file = args.stats_file
    output_file = args.output_file
    scale = args.scale

    # Load stats
    try:
        with open(stats_file, "r") as f:
            stats = json.load(f)
    except Exception as e:
        print(f"Error loading stats file: {e}", file=sys.stderr)
        sys.exit(1)

    # Load .env file from repo root if present
    env_path = os.path.join(os.path.dirname(__file__), "..", "..", ".env")
    if os.path.exists(env_path):
        with open(env_path) as f:
            for line in f:
                line = line.strip()
                if line and not line.startswith("#") and "=" in line:
                    key, value = line.split("=", 1)
                    os.environ.setdefault(key.strip(), value.strip())

    # Set up Discord resolver if token and guild_id available
    resolver = None
    token = os.environ.get("DISCORD_TOKEN")
    guild_id = os.environ.get("GUILD_ID")

    if token and guild_id:
        resolver = DiscordUserResolver(token, guild_id)
        print("Discord username and role color resolution enabled", file=sys.stderr)
    elif token:
        resolver = DiscordUserResolver(token)
        print("Discord username resolution enabled (no GUILD_ID, colors disabled)", file=sys.stderr)
    else:
        print("Warning: DISCORD_TOKEN not set, usernames will show as IDs", file=sys.stderr)

    # Render and save
    renderer = InfographicRenderer(stats, resolver, scale=scale)
    img = renderer.render()
    # Set DPI metadata (72 * scale gives effective DPI)
    dpi = 72 * scale
    img.save(output_file, "PNG", dpi=(dpi, dpi))
    print(f"Saved {img.width}x{img.height} infographic to {output_file} (scale={scale}x, {dpi} DPI)", file=sys.stderr)


if __name__ == "__main__":
    main()
