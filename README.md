# Schedule Transformer

This program fetches the Con schedule from "Guidebook" and converts
it to a JSON format for download by "watson" - a Discord bot
for conventions.

Parameters are set through environment variables:

- GB_API_KEY - the API key for Guidbook.
- GB_ID - The GuideID for Guidebook

In general if the Schedule Transformer encounters any kind of an error
it will exit with a logged message and expect a hooman to fix the issue
and run it again.
