CREATE TABLE testdata_empty (
    id SERIAL PRIMARY KEY,
    name text
);

CREATE TABLE testdata (
    id SERIAL PRIMARY KEY,
    text_val text,
    int_val integer,
    bool_val boolean,
    null_val text DEFAULT NULL
);

INSERT INTO testdata (text_val, int_val, bool_val) VALUES ('text_1', 1, true);
INSERT INTO testdata (text_val, int_val, bool_val, null_val) VALUES ('text_2', 2, false, 'null_val_2');