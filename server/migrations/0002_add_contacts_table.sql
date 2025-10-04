CREATE TABLE contacts (
  user_id INTEGER NOT NULL,
  contact_id INTEGER NOT NULL,
  PRIMARY KEY (user_id, contact_id),
  FOREIGN KEY(user_id) REFERENCES users(id),
  FOREIGN KEY(contact_id) REFERENCES users(id)
);
