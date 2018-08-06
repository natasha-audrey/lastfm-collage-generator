module.exports = (deps) => {
  const {
    axios,
    Jimp,
    moment,
  } = deps;

  const getWeeklyChartData = (req, res, next) => {
    res.locals.top_tracks = axios.get(`http://ws.audioscrobbler.com/2.0/?method=user.gettopalbums&user=n8yo&api_key=${process.env.API_KEY}&period=7day&format=json`)
      .then((response) => {
        const { data } = response;
        const albums = [];
        for (const albumData of Object.values(data.topalbums.album)) {
          const album = {
            name: albumData.name,
            artist: albumData.artist.name,
            image: albumData.image[3],
          };
          albums.push(album);
        }
        res.locals.albums = albums;
        next();
      })
      .catch((err) => {
        res.status(500).send(`Error with axios request ${err}`);
      });
  };

  const getAlbums = async (req, res, next) => {
    const collage = new Jimp(900, 900, 0xFFFFFFFF, (err) => {
      if (err) {
        console.error(err);
      }
    });
    let xPos = 0;
    let yPos = 0;

    let c = 0;
    for (let i = 0; i < 9; i++) {
      const album = res.locals.albums[i];
      Jimp.read(album.image['#text'])
        .then((image) => {
          const loadFont = Jimp.loadFont('./fonts/open-sans-16-black/open-sans-16-black.fnt')
            .then((font) => {
              image.print(font, 11, 11, `${album.artist} - ${album.name}`, 280);
            })
            .catch(err => console.error(err));

          loadFont
            .then(() => {
              Jimp.loadFont('./fonts/open-sans-16-white/open-sans-16-white.fnt')
                .then((whiteFont) => {
                  c += i;
                  image.print(whiteFont, 10, 10, `${album.artist} - ${album.name}`, 280);
                  yPos = Math.floor(i / 3) * 300;
                  xPos = (i % 3) * 300;
                  collage.composite(image, xPos, yPos);
                  collage.write(`./tmp/collage-${moment(new Date()).format('MM-DD')}.png`);
                  // This is a silly way to handle this.
                  if (c === 36) {
                    next();
                  }
                })
                .catch(err => console.error(err));
            })
            .catch(err => console.error(err));
        })
        .catch(err => console.error(err));
    }
  };

  return ({
    getWeeklyChartData,
    getAlbums,
  });
};
