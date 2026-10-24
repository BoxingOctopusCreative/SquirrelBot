UPDATE role_gates
SET kick_reason = 'Did not finish the server application in time.'
WHERE kick_reason IN (
    'Did not answer the verification questions in time',
    'Did not receive the required role in time'
);

UPDATE role_gates
SET dm_message = ''
WHERE dm_message = 'Welcome to {server}. Answer the verification questions within {duration} or you will be removed. You must be 18 or older.';
