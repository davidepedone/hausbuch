# Spec Delta

## Purpose

Brings the expiry warnings to the household's phones and computers as a daily notification, so that what starts
expiring is noticed without opening the app.

## ADDED Requirements

### Requirement: Notifications need a secure address
Notifications SHALL be offered only where passkeys are: when the installation is configured with the address its users
open and that address is HTTPS, or HTTP on `localhost`. Otherwise no page SHALL mention notifications and none SHALL be
sent.

#### Scenario: Plain HTTP installation
- **WHEN** the installation has no address configured, or an HTTP address other than `localhost`
- **THEN** the account page shows nothing about notifications

#### Scenario: HTTPS installation
- **WHEN** the installation's address is HTTPS
- **THEN** the account page has a "Notifications" section

### Requirement: Notifications turned on per device
The account page SHALL let a signed-in user turn notifications on for the browser and device they are using, after
the browser's permission, and off again. It SHALL show whether they are on for this device. Turning them on or off on
one device SHALL NOT change any other device.

#### Scenario: Turning on
- **WHEN** a user chooses "Turn on notifications on this device" and allows them in the browser
- **THEN** the page says notifications are on for this device, and this device receives the user's summaries

#### Scenario: Permission refused
- **WHEN** a user chooses "Turn on notifications on this device" and refuses the browser's permission
- **THEN** the page says the browser didn't allow notifications, and notifications stay off for this device

#### Scenario: Turning off
- **WHEN** a user with notifications on for this device chooses "Turn off notifications on this device"
- **THEN** this device no longer receives summaries, and the user's other devices still do

### Requirement: Home Screen needed on iPhone and iPad
When the browser can't receive notifications, the account page SHALL say so instead of offering to turn them on, and
SHALL explain that on iPhone and iPad hausbuch must first be added to the Home Screen and opened from there.

#### Scenario: Safari tab on iPhone
- **WHEN** a user opens the account page in a Safari tab on an iPhone, without having opened hausbuch from the Home
  Screen
- **THEN** the "Notifications" section explains how to add hausbuch to the Home Screen and has no button to turn them on

### Requirement: Choices of time and sections
The account page SHALL let a user choose the time of their daily summary and the sections it covers, among
Medicines, Pantry and Supplies. Until changed, the time SHALL be 09:00 and all three sections SHALL be covered. The
choices SHALL belong to the user and apply to all of their devices.

#### Scenario: Defaults
- **WHEN** a user turns notifications on for the first time
- **THEN** the account page shows 09:00 and Medicines, Pantry and Supplies all chosen

#### Scenario: Changed choices
- **WHEN** a user with notifications on two devices chooses 18:30 and only Medicines
- **THEN** both devices receive summaries at 18:30 about Medicines only

#### Scenario: No section chosen
- **WHEN** a user unchecks every section
- **THEN** no summary is sent to them

### Requirement: Daily summary of what is new
Once a day, at the user's chosen time in the installation's time zone, the system SHALL send the user's devices a
summary of the rows in the chosen sections that are new since their previous summary. A row SHALL be new when it is
expiring and the user was never told so, or expired and the user was never told so. When nothing is new, no summary
SHALL be sent.

#### Scenario: Something started expiring
- **WHEN** a user's time is 09:00 and the pantry's "Yogurt" became expiring yesterday
- **THEN** at 09:00 the user's devices receive a summary that lists "Yogurt"

#### Scenario: Nothing new
- **WHEN** at a user's time the only expiring row is one their previous summary already listed as expiring
- **THEN** no summary is sent to them that day

#### Scenario: Expiring, then expired
- **WHEN** a summary listed "Yogurt" as expiring, and by the next day's time it has expired
- **THEN** the next day's summary lists "Yogurt" as expired

#### Scenario: Section not chosen
- **WHEN** a user covers only Medicines and a pantry row becomes expiring
- **THEN** no summary lists that row

#### Scenario: First summary
- **WHEN** a user turns notifications on while 3 rows in their sections are already expired or expiring
- **THEN** their next summary lists all 3

### Requirement: A row told again after a change
A row that stops being expiring or expired, because its effective expiry date changed or the row was removed, SHALL be
forgotten for that user, so that it is new again if it later becomes expiring or expired.

#### Scenario: Date pushed back, then due again
- **WHEN** a summary listed "Yogurt" as expiring, its expiry date is then moved beyond the lead time, and weeks later it
  becomes expiring again
- **THEN** the summary after it becomes expiring again lists "Yogurt"

### Requirement: What a summary says
A summary SHALL say how many rows it lists and name them, expired first, then by effective expiry date, each with
its section; when they don't all fit in a notification, it SHALL name the first ones and say how many more there are.
Choosing the notification SHALL open the home page.

#### Scenario: Two rows
- **WHEN** a summary lists "Yogurt" in Pantry, expired, and "Eye drops" in Medicines, expiring
- **THEN** the notification says 2 things to check and names "Yogurt" before "Eye drops", each with its section

#### Scenario: Many rows
- **WHEN** a summary lists 40 rows
- **THEN** the notification says 40 things to check, names the first ones and says how many more there are

#### Scenario: Opening it
- **WHEN** a user chooses a summary notification on their device
- **THEN** hausbuch opens on the home page

### Requirement: Summary when the time was missed
When the system was not running at a user's time, it SHALL send that day's summary as soon as it runs again on the
same day. A user SHALL receive at most one summary a day, and a day that ended without one SHALL NOT be sent later;
what was new stays new for the next summary.

#### Scenario: Started late
- **WHEN** a user's time is 09:00, the system starts at 11:00 and a pantry row became expiring overnight
- **THEN** the user receives the summary shortly after 11:00, and no other summary that day

#### Scenario: Day missed
- **WHEN** the system didn't run at all yesterday and a row became expiring yesterday
- **THEN** today's summary lists that row, and no summary is sent for yesterday

### Requirement: Notifications stop with the device's sign-in
A device's notifications SHALL belong to its sign-in: they SHALL stop when it signs out or its session ends unused.
Disabling an account SHALL stop them on every device. A device the push service reports as no longer subscribed SHALL
be turned off.

#### Scenario: Sign out
- **WHEN** a user with notifications on signs out on their phone
- **THEN** the phone receives no more summaries, and signing in again shows notifications off on it

#### Scenario: Session ended unused
- **WHEN** a device with notifications on isn't used to open hausbuch for longer than the session timeout
- **THEN** it receives no more summaries

#### Scenario: Disabled account
- **WHEN** an account with notifications on two devices is disabled
- **THEN** neither device receives summaries

#### Scenario: Subscription gone
- **WHEN** a summary is sent to a device whose subscription the push service says has expired
- **THEN** that device is turned off and the user's other devices still receive the summary

### Requirement: Encrypted notifications
What a notification says SHALL be encrypted so that only the receiving browser can read it; the push service that
delivers it SHALL receive no row names or section names in readable form.

#### Scenario: Sent payload
- **WHEN** a summary naming "Yogurt" is sent to a device
- **THEN** the request to the push service does not contain "Yogurt" or "Pantry" in readable form

### Requirement: Summaries in the log
The log SHALL have a line for each summary sent, with the account, the number of rows and the number of devices that
received it, and a line for each device that couldn't be reached. It SHALL NOT hold row names or subscription
addresses.

#### Scenario: Summary sent
- **WHEN** a summary of 2 rows reaches a user's 2 devices
- **THEN** the log has one line with the account, 2 rows and 2 devices, and no row names
