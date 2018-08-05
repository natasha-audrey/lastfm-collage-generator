import express from 'express';
import axios from 'axios';
import Jimp from 'jimp';
import moment from 'moment';
import makeMiddleware from './middleware';

const middleware = makeMiddleware({
  axios,
  Jimp,
  moment,
});

const router = express.Router();
router.get('/', (req, res) => {
  res.status(200).send(`
  <div>
    <a href="/weeklyChart"> View Chart </a> <br>
    <a href="/generateWeeklyChart"> Make weekly chart </a>
  </div>
  `);
});
router.get('/chartJson', [
  middleware.getWeeklyChartData,
  (req, res) => res.status(200).send(res.locals.albums),
]);
router.get('/generateWeeklyChart', [
  middleware.getWeeklyChartData,
  middleware.getAlbums,
  (req, res) => {
    res.redirect('/weeklyChart');
  },
]);
router.get('/weeklyChart', (req, res) => {
  res.setHeader('Content-Type', 'image/png');
  res.sendFile(`${__dirname.replace('/server', '')}/tmp/collage-${moment(new Date()).format('MM-DD')}.png`);
});
router.get('*', (req, res) => {
  res.status(404).send('404');
});

export default router;
