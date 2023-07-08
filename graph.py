import sys
import pandas as pd
import matplotlib.pyplot as plt


class Category:
    def __init__(self, stake, start):
        self.stake = stake
        self.start = start
        self.end = 0
        self.accumulated_rewards = 0
        self.validators = []
        self.rewards = []

    def num_validator(self):
        return self.end - self.start + 1

    def expected_reward(self, number_of_days, online_stake):
        return number_of_days * 8640 * self.stake / online_stake

    def average_reward(self):
        return self.accumulated_rewards / (self.num_validator())


def draw(filename, committee_size, offline, number_of_days):
    df = pd.read_csv(filename)

    stakes = df['Stake']
    rewards = df['Reward']
    online_stake = df['Stake'].sum()
    total_stake = online_stake + (online_stake * offline/100)

    categories = []
    category = Category(stakes[0], 0)
    for num, stake in enumerate(stakes):
        if category.stake != stake:
            categories.append(category)
            category = Category(stake, num)

        category.validators.append(num+1)
        category.rewards.append(rewards[num])
        category.accumulated_rewards += rewards[num]
        category.end = num

    categories.append(category)

    # print(categories)

    colors = ['red', 'blue', 'green', 'orange', 'purple',
              'cyan', 'magenta', 'yellow', 'brown', 'gray']

    legend_text = []
    for i, cat in enumerate(categories):
        color = colors[i % len(colors)]
        x = cat.validators
        y = cat.rewards
        plt.scatter(x, y, color=color, label='Validators')

        legend_text.append(f'Rewards per {cat.stake} PAC Coin')

    # Uncomment this to see the sortitions
    #
    # x = df['Validator']
    # y = df['Sortition']
    # plt.scatter(x, y, color="black", label='Validators', marker="+")
    # legend_text.append(f'Number of evaluated Sortition')


    for i, cat in enumerate(categories):
        average = cat.average_reward()
        plt.hlines(y=average, xmin=cat.start, xmax=cat.end +
                   1, color="grey", linestyles="--")

        x = cat.start
        y = average
        plt.text(x, y, f'{y}',  va='bottom', color="grey")

        if i== 0:
            legend_text.append(f'Average rewards')

        expected = cat.expected_reward(number_of_days, online_stake)
        plt.hlines(y=expected, xmin=cat.start, xmax=cat.end +
                   1, color="black", linestyles="-")

        x = cat.start
        y = expected
        plt.text(x, y, f'{y}',  va='top', color="black")

        if i== 0:
            legend_text.append(f'Expected rewards')




    plt.legend(legend_text)

    plt.figtext(0.15, 0.15, f'Committee size: {committee_size}\nNumber of days: {number_of_days}\nOffline: {offline}%')

    plt.xlabel('Validator')
    plt.ylabel('Rewards')
    plt.tight_layout()
    plt.show()


if __name__ == '__main__':
    if len(sys.argv) < 5:
        print("Please provide CSV filename, committee size, offline percentage and number of days")
    else:
        filename = sys.argv[1]
        committee_size = float(sys.argv[2])
        offline = float(sys.argv[3])
        number_of_days = float(sys.argv[4])
        draw(filename, committee_size, offline, number_of_days)
