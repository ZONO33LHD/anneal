const express = require('express');
const _ = require('lodash');
const axios = require('axios');

const app = express();

app.get('/users', async (req, res) => {
  const { data } = await axios.get('https://example.com/api/users');
  const sorted = _.sortBy(data, 'name');
  res.json(sorted);
});

app.listen(3000, () => {
  console.log('listening on 3000');
});

module.exports = app;
