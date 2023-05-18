const crowdin = require('@crowdin/crowdin-api-client');
const personalToken = process.env.CROWDIN_PERSONAL_TOKEN;
const projectId = '574591';

/**
 * Initialization of crowdin client
 * @return {object} crowdin client
 */
const initClient = () => {
  if (!personalToken) {
    console.warn(
      'No crowding personal token, some features might not work as expected'
    );
    return null;
  }

  return new crowdin.default({
    token: personalToken
  });
};

/**
 * Get translation progress
 * @return {object} translation progress
 */
async function getTranslationProgress() {
  let translationProgress = {};
  const { translationStatusApi } = initClient() || {};

  if (!translationStatusApi) {
    return translationProgress;
  }

  await translationStatusApi
    .getProjectProgress(projectId)
    .then((res) => {
      res.data.forEach((item) => {
        translationProgress[item.data.languageId] = item.data.approvalProgress;
      });
    })
    .catch((err) => {
      console.error(err);
    });

  return translationProgress;
}

module.exports = {
  getTranslationProgress
};
