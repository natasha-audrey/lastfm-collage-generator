import app from './app';

app.listen(process.env.PORT || 3000, () => {
  console.log('Port: ', (process.env.PORT || 3000));
});
