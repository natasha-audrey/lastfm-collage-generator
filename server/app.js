import express from 'express';
import dotenv from 'dotenv';
import router from './router';

dotenv.config();


const app = express();
app.get('*', router);

export default app;
