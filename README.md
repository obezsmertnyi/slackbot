# Slackbot. Автоматизація оновлень версій аплікацій у кластері Kubernetes за допомогою команд Slack-боту

Slackbot призначений для автоматизації процесу оновлень версій аплікацій у кластері Kubernetes за допомогою команд у месенджері Slack у захищених спеціально налаштованих каналах.
Бот дозволяє веріфікованому розробнику у відповідному Slack-каналі робити наступні операції:

- перегляд поточного стану версій аплікацій розгорнутих у Kubernetes для різних оточень
- порівняння версій аплікацій у розрізі усіх наявних оточень кластеру
- автоматичне розгортання нових версій аплікацій
- автоматичний відкат до попередньої версії аплікації

## Команди Slackbot

1) /list {dev, qa, stage, prod} - команда отримання поточного стану розгорнутих версій аплікацій для кожного середовища

![1_List_command_Slackbot](https://github.com/sbazanov/InfiniteLoopBreakers/assets/96147501/882d41f5-0ee2-4205-9edb-18392f77125e)

2) /diff {app_name} - команда отримання розгорнутих версій в усіх наявних оточеннях кластеру конкретного аплікейшену

![2_Diff_command_Slackbot](https://github.com/sbazanov/InfiniteLoopBreakers/assets/96147501/9536cc07-a673-4e28-be32-335c4915c17d)

3) /promote {dev, qa, stage, prod} {app_name} - команда автоматичного деплою нової версії аплікації у вказаному середовищі

![3_Promote_command_Slackbot](https://github.com/sbazanov/InfiniteLoopBreakers/assets/96147501/172a0502-0b83-4057-9e0f-21df11d88758)

4) /rollback {dev, qa, stage, prod} {app_name} - команда автоматичного відкату до попередньої версії аплікації

![4_Rollback_command_Slackbot](https://github.com/sbazanov/InfiniteLoopBreakers/assets/96147501/f555b886-fa1f-427c-a47c-f58fa8408713)
   
Зазначимо наступне: 
- {app_name} - це {label} подів у кластері Kubernetes 
- {dev, qa, stage, prod} - це відокремлені неймспейси


## Виняткові ситуації

1) /list. Введення неіснуючого середовища

![5_List_not_correct_env_Slackbot](https://github.com/sbazanov/InfiniteLoopBreakers/assets/96147501/bf40fa5e-1796-4f88-a7d1-ef8bee3c3df7)

2) /promote. Спроба оновити повторно актуальний додаток

![6_Promote_not_needed_Slackbot](https://github.com/sbazanov/InfiniteLoopBreakers/assets/96147501/75f288a9-f447-4f4c-aefa-2afb9c21b7c6)

3) /rollback. Спроба відкатити на попередню версію після 1-ого деплою, коли у базі ще не зафіксована попередня версія

![7_Rollback_no_previous_version_Slackbot](https://github.com/sbazanov/InfiniteLoopBreakers/assets/96147501/0cc0f145-a6d4-4825-82d8-a9ae15f0813b)
