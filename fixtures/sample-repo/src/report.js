const chalk = require('chalk');
const _ = require('lodash');

function printReport(items) {
  const grouped = _.groupBy(items, 'status');
  for (const [status, group] of Object.entries(grouped)) {
    console.log(chalk.bold(status), group.length);
  }
}

module.exports = { printReport };
