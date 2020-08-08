-- Using same table as https://cloud.google.com/spanner/docs/quickstart-console
-- Note: change SingerId as string to avoid hotspot

CREATE TABLE Singers (
  SingerId   STRING(MAX) NOT NULL,
  FirstName  STRING(1024),
  LastName   STRING(1024),
  SingerInfo BYTES(MAX),
  BirthDate  DATE,
) PRIMARY KEY(SingerId);