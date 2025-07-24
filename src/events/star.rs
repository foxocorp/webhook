use crate::errors::Error;
use crate::events::Event;
use crate::events::base::{Repository, User, WebhookMessage};
use serde::Deserialize;

#[derive(Deserialize)]
pub struct StarEvent {
    pub action: String,
    pub sender: User,
    pub repository: Repository,
}

impl Event for StarEvent {
    fn handle(&self) -> Result<Option<WebhookMessage>, Error> {
        if self.repository.private {
            return Ok(None);
        }

        Ok(Some(WebhookMessage {
            content: format!(
                "[{}](<{}>) starred [{}](<{}>) {}",
                self.sender.login,
                self.sender.html_url,
                self.repository.name,
                self.repository.html_url,
                std::env::var("HAPPY_EMOJI").unwrap_or_else(|_| "🐈".into())
            ),
            username: self.sender.login.clone(),
            avatar_url: self.sender.avatar_url.clone(),
        }))
    }
}
