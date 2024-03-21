# graylog-to-jira

## Purpose

Logs stored in a system like Graylog can be helpful to reference from bug reports in Jira but simply adding a link to a Graylog search URL may not be helpful in cases where the logs have exceeded the Graylog retention limit by the time the bug is picked up.  This tool is intended to make it easy to export the results of a Graylog search and attach them to a Jira.
