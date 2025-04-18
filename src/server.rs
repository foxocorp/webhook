use crate::GITHUB_EVENT;
use crate::parser::parse_event;
use crate::events::base::Credentials;
use actix_web::http::StatusCode;
use actix_web::{HttpRequest, HttpResponse, Responder, post, web};
use log::error;

#[post("/{id}/{token}")]
async fn webhook(
    req: HttpRequest,
    creds: web::Path<Credentials>,
    body: web::Bytes,
) -> impl Responder {
    let event = req
        .headers()
        .get(GITHUB_EVENT)
        .and_then(|v| v.to_str().ok())
        .unwrap_or("")
        .to_string();
    if creds.id.trim().is_empty() || creds.token.trim().is_empty() || event.trim().is_empty() {
        return HttpResponse::Ok().body("Header or credentials are empty");
    }

    tokio::spawn(async move {
        if let Err(e) = parse_event(event, body, creds.into_inner()).await {
            error!("{}", e);
        }
    });

    HttpResponse::Ok().status(StatusCode::NO_CONTENT).finish()
}
