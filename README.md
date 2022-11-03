<p align="center">
  <h1>ZENFLOWS INBOX</h1>

  Simple **HTTP inbox** for [interfacer-gui](https://github.com/dyne/interfacer-gui/) 
</p>

# Deployment
For the deployment see the subdirectory `devops`, there is an `ansible` role. It is also available a `Dockerfile` and a `docker-compose.yml`.

# Usage
The HTTP service provides two URLs:
- `/send`: a `POST` request where the body is a json with three fields: `sender`, `receivers` and `content`. The `sender` is the ID (of the agent in zenflows as string) and `receivers` is a list of the IDs of the agent (as strings) that should receive the message. The `content` is saved as JSON inside a postgresql field, when an agent want to see his messages has to make a call to `read`;
- `/read`: a `POST` request where the body is a json with exactly two fields: `request_id` and `receiver`. The `receiver` is the ID (as string) of the agent we want to read the messages of, while `request_id` is a random value, in the response the `inbox` service will put the same value in the `receiver_id`. There could be a third field `only_unread` that return only the messages for which the `read` flag is `false`;
- `set-read`: a `POST` request where the body is a JSON with three fields: `message_id`, `receiver`, `read`. The `message_id` and `receiver` determine the message we are talking about and which receiver. The `read` flag tells if the `receiver` has read (or not) the message. (Implement the read/unread feature);
- `count-unread`: a `POST` request where the body is a JSON with one field: `receiver`. It returns the number of messages with the `read` flag set to false.


All request have to be signed with the private key of the `sender` (in `/send`) or `receiver` (in `/read`) agent with [zenflows-crypto](https://github.com/dyne/zenflows-crypto.git) and the signature has to be put in the HTTP request in the header `zenflows-sign`.

# Examples
See subdirectory `examples`
