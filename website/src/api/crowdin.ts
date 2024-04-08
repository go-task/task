import crowdin, { Credentials, TranslationStatusModel, ResponseObject, LanguagesModel, Languages } from '@crowdin/crowdin-api-client';

const projectId = 574591;
const credentials: Credentials = {
  token: process.env.CROWDIN_PERSONAL_TOKEN,
};

// This adds the language field to LanguageProgress which is missing in the original model
export interface LanguageProgress extends TranslationStatusModel.LanguageProgress {
  language: LanguagesModel.Language;
}

const initClient = () => {
  if (credentials.token === '') {
    console.warn(
      'No crowdin personal token, some features might not work as expected'
    );
    return null;
  }
  return new crowdin(credentials);
};

export async function getTranslationProgress(): Promise<Map<string, LanguageProgress>> {
  var progress = new Map<string, LanguageProgress>();
  await initClient().translationStatusApi
    .getProjectProgress(projectId)
    .then((res) => {
      res.data.forEach((item: ResponseObject<LanguageProgress>) => {
        progress.set(item.data.language.id, item.data);
      });
    })
    .catch((err) => {
      console.error(err);
    });
  return progress;
}
