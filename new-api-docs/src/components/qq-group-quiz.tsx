'use client';

import { useState } from 'react';

interface QuizProps {
  locale?: 'en' | 'zh' | 'ja';
}

const translations = {
  en: {
    questions: [
      {
        id: 'q1',
        question: '1. What is New API?',
        options: [
          { value: 'a', label: 'A commercial API sales platform' },
          {
            value: 'b',
            label:
              'An open-source AI interface management and distribution system',
          },
          { value: 'c', label: 'A paid software' },
          { value: 'd', label: 'A public welfare API site' },
        ],
        correct: 'b',
        errorMessage:
          'Please re-read the project introduction. New API is an open-source AI interface management and distribution system.',
      },
      {
        id: 'q2',
        question:
          '2. Is it allowed to post information about buying accounts or API products in the group?',
        options: [
          { value: 'true', label: 'Yes' },
          { value: 'false', label: 'No' },
        ],
        correct: 'false',
        errorMessage:
          'According to Group Rule 2: The group chat prohibits posting any information related to account or API product sales or purchases.',
      },
      {
        id: 'q3',
        question: '3. Does New API have a closed-source paid version?',
        options: [
          {
            value: 'true',
            label: 'Yes, there is a closed-source paid version',
          },
          {
            value: 'false',
            label:
              'No, New API is open-source software, but a commercial license is required in certain scenarios',
          },
        ],
        correct: 'false',
        errorMessage:
          'According to Group Rule 4: New API is open-source software, but a commercial license is required in certain scenarios. For details, please see the Project Introduction.',
      },
      {
        id: 'q4',
        question:
          '4. Are the group owner and administrators obligated to provide me with technical support?',
        options: [
          {
            value: 'true',
            label: 'Yes, they are obligated to provide technical support',
          },
          {
            value: 'false',
            label:
              'No, they are not obligated to provide technical support; an issue should be submitted',
          },
        ],
        correct: 'false',
        errorMessage:
          'According to Group Rule 1: Group administrators and the group owner are under no obligation to provide you with any technical support. If you have questions, please submit an issue.',
      },
      {
        id: 'q5',
        question:
          '5. Can I purchase API products sold by administrators in the group?',
        options: [
          { value: 'true', label: 'Yes, I can purchase' },
          {
            value: 'false',
            label: 'No, this group does not sell any API products',
          },
        ],
        correct: 'false',
        errorMessage:
          'According to Group Rule 3: This group does not sell any API products. Please do not trust or purchase API products from anyone (including administrators).',
      },
    ],
    submitButton: 'Submit Answers',
    answerAllQuestions: 'Please answer all questions before submitting.',
    successTitle: 'Congratulations, verification passed!',
    successMessage:
      'Thank you for carefully reading the group rules. Welcome to our community!',
    methodOne: 'Method One: Scan QR Code',
    methodTwo: 'Method Two: Click Link',
    joinLink: 'Click here to join the QQ group directly',
    incorrectAnswer: 'Incorrect Answer!',
  },
  zh: {
    questions: [
      {
        id: 'q1',
        question: '1. New API 是什么？',
        options: [
          { value: 'a', label: '一个商业API销售平台' },
          { value: 'b', label: '一个开源的AI接口管理和分发系统' },
          { value: 'c', label: '一个付费软件' },
          { value: 'd', label: '一个公益API站点' },
        ],
        correct: 'b',
        errorMessage:
          '请重新阅读项目介绍。New API 是一个开源的AI接口管理和分发系统。',
      },
      {
        id: 'q2',
        question: '2. 群内是否允许发布买卖账号或API产品的信息？',
        options: [
          { value: 'true', label: '是' },
          { value: 'false', label: '否' },
        ],
        correct: 'false',
        errorMessage:
          '根据群规第2条：群聊禁止发布任何与账号或API产品买卖相关的信息。',
      },
      {
        id: 'q3',
        question: '3. New API 是否有闭源的付费版本？',
        options: [
          { value: 'true', label: '有，存在付费的闭源版本' },
          {
            value: 'false',
            label: '没有，New API 采用 AGPLv3 开源协议，遵守协议即可免费使用',
          },
        ],
        correct: 'false',
        errorMessage:
          '根据群规第4条：New API 采用 GNU AGPLv3 开源协议，只要遵守开源协议即可免费使用。',
      },
      {
        id: 'q4',
        question: '4. 群主和管理员是否有义务为我提供技术支持？',
        options: [
          { value: 'true', label: '是，他们有义务提供技术支持' },
          {
            value: 'false',
            label: '否，他们没有义务提供技术支持，应该提交issue',
          },
        ],
        correct: 'false',
        errorMessage:
          '根据群规第1条：群管理员和群主没有义务为您提供任何技术支持。如有问题，请提交issue。',
      },
      {
        id: 'q5',
        question: '5. 我可以购买群内管理员出售的API产品吗？',
        options: [
          { value: 'true', label: '可以购买' },
          { value: 'false', label: '不可以，本群不出售任何API产品' },
        ],
        correct: 'false',
        errorMessage:
          '根据群规第3条：本群不出售任何API产品。请勿相信或购买任何人（包括管理员）出售的API产品。',
      },
    ],
    submitButton: '提交答案',
    answerAllQuestions: '请回答所有问题后再提交。',
    successTitle: '恭喜，验证通过！',
    successMessage: '感谢您仔细阅读群规。欢迎加入我们的社区！',
    methodOne: '方式一：扫描二维码',
    methodTwo: '方式二：点击链接',
    joinLink: '点击这里直接加入QQ群',
    incorrectAnswer: '答案错误！',
  },
  ja: {
    questions: [
      {
        id: 'q1',
        question: '1. New APIとは何ですか？',
        options: [
          { value: 'a', label: '商用API販売プラットフォーム' },
          {
            value: 'b',
            label: 'オープンソースのAIインターフェース管理・配布システム',
          },
          { value: 'c', label: '有料ソフトウェア' },
          { value: 'd', label: '公益APIサイト' },
        ],
        correct: 'b',
        errorMessage:
          'プロジェクト紹介を再度お読みください。New APIはオープンソースのAIインターフェース管理・配布システムです。',
      },
      {
        id: 'q2',
        question:
          '2. グループ内でアカウントやAPI製品の売買情報を投稿することは許可されていますか？',
        options: [
          { value: 'true', label: 'はい' },
          { value: 'false', label: 'いいえ' },
        ],
        correct: 'false',
        errorMessage:
          'グループルール2により：グループチャットでは、アカウントやAPI製品の売買に関する情報の投稿は禁止されています。',
      },
      {
        id: 'q3',
        question: '3. New APIにはクローズドソースの有料版がありますか？',
        options: [
          { value: 'true', label: 'はい、クローズドソースの有料版があります' },
          {
            value: 'false',
            label:
              'いいえ、New APIはオープンソースソフトウェアですが、特定のシナリオでは商用ライセンスが必要です',
          },
        ],
        correct: 'false',
        errorMessage:
          'グループルール4により：New APIはオープンソースソフトウェアですが、特定のシナリオでは商用ライセンスが必要です。詳細については、プロジェクト紹介をご覧ください。',
      },
      {
        id: 'q4',
        question:
          '4. グループオーナーと管理者は技術サポートを提供する義務がありますか？',
        options: [
          {
            value: 'true',
            label: 'はい、技術サポートを提供する義務があります',
          },
          {
            value: 'false',
            label: 'いいえ、義務はありません。issueを提出してください',
          },
        ],
        correct: 'false',
        errorMessage:
          'グループルール1により：グループ管理者とグループオーナーは、技術サポートを提供する義務はありません。ご質問がある場合は、issueを提出してください。',
      },
      {
        id: 'q5',
        question: '5. グループ内で管理者が販売するAPI製品を購入できますか？',
        options: [
          { value: 'true', label: 'はい、購入できます' },
          {
            value: 'false',
            label: 'いいえ、このグループではAPI製品を販売していません',
          },
        ],
        correct: 'false',
        errorMessage:
          'グループルール3により：このグループではAPI製品を販売していません。誰からも（管理者を含む）API製品を信用したり購入したりしないでください。',
      },
    ],
    submitButton: '回答を送信',
    answerAllQuestions: 'すべての質問に回答してから送信してください。',
    successTitle: 'おめでとうございます、認証に合格しました！',
    successMessage:
      'グループルールを注意深くお読みいただきありがとうございます。コミュニティへようこそ！',
    methodOne: '方法1：QRコードをスキャン',
    methodTwo: '方法2：リンクをクリック',
    joinLink: 'ここをクリックしてQQグループに直接参加',
    incorrectAnswer: '不正解です！',
  },
};

export function QQGroupQuiz({ locale = 'en' }: QuizProps) {
  const t = translations[locale] || translations.en;
  const [answers, setAnswers] = useState<Record<string, string>>({});
  const [error, setError] = useState<string | null>(null);
  const [showResult, setShowResult] = useState(false);
  const [isShaking, setIsShaking] = useState(false);

  const handleAnswerChange = (questionId: string, value: string) => {
    setAnswers((prev) => ({ ...prev, [questionId]: value }));
    setError(null);
  };

  const handleSubmit = () => {
    // Check if all questions are answered
    const unanswered = t.questions.find((q) => !answers[q.id]);
    if (unanswered) {
      setError(t.answerAllQuestions);
      return;
    }

    // Check answers
    const wrongQuestion = t.questions.find((q) => answers[q.id] !== q.correct);
    if (wrongQuestion) {
      setError(wrongQuestion.errorMessage);
      setIsShaking(true);
      setTimeout(() => setIsShaking(false), 500);
      return;
    }

    // All correct!
    setShowResult(true);
  };

  if (showResult) {
    return (
      <>
        <div className="bg-fd-card text-fd-card-foreground my-4 flex flex-row gap-2 rounded-xl border p-3 ps-1 text-sm shadow-md">
          <div
            role="none"
            className="bg-fd-primary w-0.5 shrink-0 rounded-sm"
          />
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width="20"
            height="20"
            viewBox="0 0 24 24"
            fill="currentColor"
            className="fill-fd-primary text-fd-card size-5 shrink-0"
          >
            <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z" />
          </svg>
          <div className="flex min-w-0 flex-1 flex-col gap-2">
            <span className="font-medium">{t.successTitle}</span>
            <span className="text-fd-muted-foreground">{t.successMessage}</span>
          </div>
        </div>

        <h3>{t.methodOne}</h3>
        <img
          src="/assets/qq_5.jpg"
          alt="QQ Group QR Code"
          style={{
            maxWidth: '300px',
            borderRadius: '8px',
            boxShadow: '0 2px 8px rgba(0,0,0,0.1)',
          }}
        />

        <h3 className="mt-5">{t.methodTwo}</h3>
        <a
          href="https://qm.qq.com/q/tTEdGHexck"
          target="_blank"
          rel="noopener noreferrer"
          className="inline-block rounded bg-blue-600 px-5 py-2.5 font-medium text-white no-underline transition-colors hover:bg-blue-700"
        >
          {t.joinLink}
        </a>
      </>
    );
  }

  return (
    <div style={{ margin: '20px 0' }}>
      {t.questions.map((question) => (
        <div
          key={question.id}
          style={{
            marginBottom: '20px',
            padding: '15px',
            background: 'var(--fd-secondary)',
            borderRadius: '4px',
          }}
        >
          <div style={{ fontWeight: 600, marginBottom: '10px' }}>
            {question.question}
          </div>
          {question.options.map((option) => (
            <label
              key={option.value}
              style={{
                display: 'block',
                margin: '8px 0',
                cursor: 'pointer',
              }}
            >
              <input
                type="radio"
                name={question.id}
                value={option.value}
                checked={answers[question.id] === option.value}
                onChange={(e) =>
                  handleAnswerChange(question.id, e.target.value)
                }
                style={{ marginRight: '8px' }}
              />
              <span>{option.label}</span>
            </label>
          ))}
        </div>
      ))}

      {error && (
        <div className="bg-fd-card text-fd-card-foreground my-4 flex flex-row gap-2 rounded-xl border p-3 ps-1 text-sm shadow-md">
          <div
            role="none"
            className="w-0.5 shrink-0 rounded-sm"
            style={{ backgroundColor: '#ef4444' }}
          />
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width="20"
            height="20"
            viewBox="0 0 24 24"
            fill="#ef4444"
            className="size-5 shrink-0"
          >
            <path d="M12 2C6.47 2 2 6.47 2 12s4.47 10 10 10 10-4.47 10-10S17.53 2 12 2zm5 13.59L15.59 17 12 13.41 8.41 17 7 15.59 10.59 12 7 8.41 8.41 7 12 10.59 15.59 7 17 8.41 13.41 12 17 15.59z" />
          </svg>
          <div className="flex min-w-0 flex-1 flex-col gap-2">
            <span className="font-medium">{t.incorrectAnswer}</span>
            <span className="text-fd-muted-foreground">{error}</span>
          </div>
        </div>
      )}

      <button
        onClick={handleSubmit}
        style={{
          width: '100%',
          padding: '12px 20px',
          background: isShaking ? '#c62828' : '#1976d2',
          color: 'white',
          border: 'none',
          borderRadius: '4px',
          fontSize: '15px',
          fontWeight: 500,
          cursor: 'pointer',
          transition: 'background 0.3s',
          animation: isShaking ? 'shake 0.5s' : 'none',
        }}
        onMouseEnter={(e) => {
          if (!isShaking) {
            (e.target as HTMLButtonElement).style.background = '#1565c0';
          }
        }}
        onMouseLeave={(e) => {
          if (!isShaking) {
            (e.target as HTMLButtonElement).style.background = '#1976d2';
          }
        }}
      >
        {t.submitButton}
      </button>

      <style>{`
        @keyframes shake {
          0%, 100% { transform: translateX(0); }
          10%, 30%, 50%, 70%, 90% { transform: translateX(-5px); }
          20%, 40%, 60%, 80% { transform: translateX(5px); }
        }
      `}</style>
    </div>
  );
}
