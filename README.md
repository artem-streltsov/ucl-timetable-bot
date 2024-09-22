# UCL Timetable Bot

The UCL Timetable Bot is a Telegram bot that helps UCL students manage their lecture schedules by providing easy access to their timetables and sending timely notifications.

## Getting Started

1. **Get your WebCal link:**
   - Log in to Portico
   - Go to My Studies -> Timetable
   - Click on "Add to Calendar"
   - Copy the WebCal URL

2. **Start using the bot:**
   - Open Telegram
   - Search for `@ucl_timetable_bot`
   - Start a chat with the bot
   - Use the `/start` command to begin

3. **Set up your timetable:**
   - Send your WebCal URL to the bot when prompted

## Available Commands

- `/start`: Begin interaction with the bot and set up your timetable
- `/set_webcal`: Set and update your WebCal link
- `/today`: Get today's lecture schedule
- `/week`: Get this week's lecture schedule
- `/settings`: View and update your notification settings
- `/set_daily_time`: Set the time for daily notifications
- `/set_weekly_time`: Set the day and time for weekly notifications
- `/set_reminder_offset`: Set how many minutes before a lecture you want to be reminded

## Configuring Notifications

The bot offers three types of notifications:

1. **Daily Summary**: A daily overview of your lectures
2. **Weekly Summary**: A weekly overview of your lectures
3. **Lecture Reminders**: Notifications before each lecture

To configure these notifications:

1. Use `/settings` to view your current notification settings
2. Use `/set_daily_time` to set when you receive daily summaries
3. Use `/set_weekly_time` to set when you receive weekly summaries
4. Use `/set_reminder_offset` to set how far in advance you want lecture reminders

## Time Zone Information

**Important:** The UCL Timetable Bot operates using UK time (either GMT or BST, depending on the time of year). This means:

- All times displayed in notifications and summaries are in UK time.
- When setting notification times, please use UK time.

Please keep this in mind when interacting with the bot, especially if you are in a different time zone.
