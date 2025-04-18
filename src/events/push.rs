use crate::errors::Error;
use crate::events::Event;
use crate::events::base::{PushCommit, Repository, User, WebhookMessage};
use regex::Regex;
use serde::Deserialize;

#[derive(Deserialize)]
pub struct PushEvent {
    pub commits: Vec<PushCommit>,
    #[serde(rename = "ref")]
    pub ref_: String,
    pub pusher: User,
    pub sender: User,
    pub repository: Repository,
}

impl Event for PushEvent {
    fn handle(&self) -> Result<Option<WebhookMessage>, Error> {
        let newline_re = Regex::new(r"(?m)^\s*\n")?;
        let link_re = Regex::new(r"\[([^]]+)]\((https?://[^)]+)\)")?;
        let mut commits = String::new();

        for c in &self.commits {
            let clean_msg = newline_re.replace_all(&c.message, "").to_string();
            let updated_msg = link_re.replace_all(&clean_msg, |caps: &regex::Captures| {
                format!("[{}](<{}>)", &caps[1], &caps[2])
            });

            commits.push_str(&format!(
                "[`{}`](<{}>) {}\n",
                &c.id[..7],
                c.url,
                updated_msg
            ));
        }

        let branch = self.ref_.strip_prefix("refs/heads/").unwrap_or(&self.ref_);
        let footer = format!(
            "\n- [{}](<{}>) on [{}](<{}>)/[{}](<{}>)",
            self.pusher.name,
            self.sender.html_url,
            self.repository.name,
            self.repository.html_url,
            branch,
            format!("{}/tree/{}", self.repository.html_url, branch),
        );

        let limit = 2000 - (footer.chars().count() + "...".len() + 1);
        if commits.chars().count() > limit {
            let mut truncated = commits.chars().take(limit).collect::<String>() + "...";
            if !truncated.ends_with(">)") {
                let lines: Vec<&str> = truncated.lines().collect();
                truncated = lines[..lines.len().saturating_sub(1)].join("\n");
            }
            commits = truncated + "\n";
        }

        Ok(Some(WebhookMessage {
            content: format!("{}{}", commits, footer),
            username: self.pusher.name.clone(),
            avatar_url: self.sender.avatar_url.clone(),
        }))
    }
}
